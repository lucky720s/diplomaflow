const fs = require("fs");
const path = require("path");

const DEFAULT_EXCLUDE_DIRS = new Set([
    ".git",
    "node_modules",
    "vendor",
    "txt", // папка с результатами
]);

const DEFAULT_EXCLUDE_FILES = new Set([
    "collected.txt",
    "diplomaflow_deploy",
    "diplomaflow_deploy.pub",
]);

const DEFAULT_EXCLUDE_EXT = new Set([
    ".exe",
]);

function normalizePath(p) {
    return p.split(path.sep).join("/");
}

function isHidden(name) {
    return name.startsWith(".");
}

function shouldSkipByName(name) {
    return DEFAULT_EXCLUDE_FILES.has(name);
}

function shouldSkipByExt(name) {
    const ext = path.extname(name);
    if (!ext) return false;
    return DEFAULT_EXCLUDE_EXT.has(ext);
}

function shouldSkipBySuffix(name) {
    return (
        name.endsWith(".pb.go") ||
        name.endsWith("validate.go") ||
        name.endsWith(".mod") ||
        name.endsWith(".sum") ||
        name.endsWith(".work") ||
        name.endsWith(".js") ||
        name.endsWith(".txt")
    );
}

async function safeReadUtf8(filePath) {
    try {
        return await fs.promises.readFile(filePath, "utf8");
    } catch {
        return null;
    }
}

async function collectAllFiles(baseDir) {
    const results = [];

    async function walk(currentPath) {
        let items;
        try {
            items = await fs.promises.readdir(currentPath, { withFileTypes: true });
        } catch {
            return;
        }

        for (const item of items) {
            const name = item.name;

            if (isHidden(name)) continue;

            const fullPath = path.join(currentPath, name);

            let stat;
            try {
                stat = await fs.promises.stat(fullPath);
            } catch {
                continue;
            }

            if (stat.isDirectory()) {
                if (DEFAULT_EXCLUDE_DIRS.has(name)) continue;
                await walk(fullPath);
                continue;
            }

            if (!stat.isFile()) continue;

            if (shouldSkipByName(name)) continue;
            if (shouldSkipByExt(name)) continue;
            if (shouldSkipBySuffix(name)) continue;

            const content = await safeReadUtf8(fullPath);
            if (content == null) continue;

            results.push({
                absPath: fullPath,
                relPath: normalizePath(path.relative(baseDir, fullPath)),
                content,
            });
        }
    }

    await walk(baseDir);
    return results;
}

async function listCmdServices(baseDir) {
    const cmdDir = path.join(baseDir, "cmd");
    let items = [];
    try {
        items = await fs.promises.readdir(cmdDir, { withFileTypes: true });
    } catch {
        return [];
    }

    return items
        .filter((d) => d.isDirectory() && !isHidden(d.name))
        .map((d) => d.name)
        .filter((name) => name !== "migrate");
}

function serviceToModule(serviceName) {
    if (serviceName === "api_gateway") return "gateway";
    if (serviceName.endsWith("_service"))
        return serviceName.replace(/_service$/, "");
    return null;
}

function makeServiceMatcher(serviceName) {
    const moduleName = serviceToModule(serviceName);

    const cmdPrefix = normalizePath(path.join("cmd", serviceName)) + "/";
    const internalPrefix = moduleName
        ? normalizePath(path.join("internal", moduleName)) + "/"
        : null;
    const protoPrefix = moduleName
        ? normalizePath(path.join("api", "proto", moduleName)) + "/"
        : null;

    return (relPath) => {
        if (relPath.startsWith(cmdPrefix)) return true;
        if (internalPrefix && relPath.startsWith(internalPrefix)) return true;
        if (protoPrefix && relPath.startsWith(protoPrefix)) return true;
        return false;
    };
}

function makeMigrationsMatcher() {
    const migPrefix = "db/migrations/";
    const cmdMigratePrefix = "cmd/migrate/";
    return (relPath) =>
        relPath.startsWith(migPrefix) ||
        relPath.startsWith(cmdMigratePrefix);
}

function makeArchitectureMatcher() {
    const prefixes = [
        "cd/",
        "ci/",
        "docker/",
        "compose/",
    ];

    const singleFiles = new Set([
        "deploy.sh",
        "docker-compose.yml",
        "docker-compose.yaml",
        "Makefile",
    ]);

    return (relPath) => {
        if (singleFiles.has(relPath)) return true;
        return prefixes.some((p) => relPath.startsWith(p));
    };
}

function formatBundle(title, files) {
    let out = "";
    out += `=== ${title} ===\n`;
    out += `FILES: ${files.length}\n\n`;

    for (const f of files) {
        out += `===== FILE START =====\n`;
        out += `PATH: ${f.relPath}\n`;
        out += `===== FILE CONTENT =====\n`;
        out += `${f.content}\n`;
        out += `===== FILE END =====\n`;
        out += `SIZE: ${Buffer.byteLength(f.content, "utf8")} bytes\n\n`;
    }

    return out;
}

async function ensureEmptyDir(dir) {
    await fs.promises.mkdir(dir, { recursive: true });
    const items = await fs.promises.readdir(dir);
    for (const name of items) {
        const p = path.join(dir, name);
        await fs.promises.rm(p, { recursive: true, force: true });
    }
}

(async () => {
    const baseDir = path.resolve("./");
    const outDir = path.join(baseDir, "txt");

    const allFiles = await collectAllFiles(baseDir);
    const services = await listCmdServices(baseDir);

    await ensureEmptyDir(outDir);

    const usedPaths = new Set();

    // migrations
    {
        const matcher = makeMigrationsMatcher();
        const files = allFiles.filter((f) => matcher(f.relPath));
        files.forEach((f) => usedPaths.add(f.relPath));

        const body = formatBundle("migrations", files);
        await fs.promises.writeFile(
            path.join(outDir, "migrations.txt"),
            body,
            "utf8"
        );
    }

    // services
    for (const svc of services) {
        const matcher = makeServiceMatcher(svc);
        const files = allFiles.filter((f) => matcher(f.relPath));
        files.forEach((f) => usedPaths.add(f.relPath));

        const body = formatBundle(svc, files);
        await fs.promises.writeFile(
            path.join(outDir, `${svc}.txt`),
            body,
            "utf8"
        );
    }

    // architecture
    {
        const matcher = makeArchitectureMatcher();
        const files = allFiles.filter((f) => matcher(f.relPath));
        files.forEach((f) => usedPaths.add(f.relPath));

        const body = formatBundle("architecture", files);
        await fs.promises.writeFile(
            path.join(outDir, "architecture.txt"),
            body,
            "utf8"
        );
    }

    // others
    {
        const files = allFiles.filter((f) => !usedPaths.has(f.relPath));
        const body = formatBundle("others", files);
        await fs.promises.writeFile(
            path.join(outDir, "others.txt"),
            body,
            "utf8"
        );
    }

    // index
    const indexLines = [];
    indexLines.push("=== generated txt bundles ===");
    indexLines.push(`baseDir: ${normalizePath(baseDir)}`);
    indexLines.push(`outDir: ${normalizePath(outDir)}`);
    indexLines.push("");
    indexLines.push(`services (${services.length}):`);
    for (const svc of services)
        indexLines.push(`- ${svc} -> ${svc}.txt`);
    indexLines.push("");
    indexLines.push("- migrations -> migrations.txt");
    indexLines.push("- architecture -> architecture.txt");
    indexLines.push("- others -> others.txt");

    await fs.promises.writeFile(
        path.join(outDir, "_index.txt"),
        indexLines.join("\n") + "\n",
        "utf8"
    );

    console.log("Готово! Сгенерировано в папке:", outDir);
})();
