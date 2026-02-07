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
    "diplomaflow_deploy",      // PRIVATE KEY !!! [[1]]
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
    if (DEFAULT_EXCLUDE_FILES.has(name)) return true;
    return false;
}

function shouldSkipByExt(name) {
    const ext = path.extname(name);
    if (!ext) return false;
    return DEFAULT_EXCLUDE_EXT.has(ext);
}

function shouldSkipBySuffix(name) {
    return (
        name.endsWith(".pb.go") ||     // generated
        name.endsWith("validate.go") || // generated
        name.endsWith(".mod") ||
        name.endsWith(".sum") ||
        name.endsWith(".work") ||
        name.endsWith(".js") ||       // чтобы не тащить сам скрипт в результаты
        name.endsWith(".txt")         // чтобы не читать output
    );
}

async function safeReadUtf8(filePath) {
    try {
        return await fs.promises.readFile(filePath, "utf8");
    } catch {
        return null; // бинарник/нечитаемо/нет доступа
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
        .filter((name) => name !== "migrate"); // migrate обработаем отдельно как migrations.txt
}

function serviceToModule(serviceName) {
    if (serviceName === "api_gateway") return "gateway";
    if (serviceName.endsWith("_service")) return serviceName.replace(/_service$/, "");
    return null;
}

function makeServiceMatcher(baseDir, serviceName) {
    const moduleName = serviceToModule(serviceName);

    const cmdPrefix = normalizePath(path.join("cmd", serviceName)) + "/";
    const internalPrefix = moduleName ? normalizePath(path.join("internal", moduleName)) + "/" : null;
    const protoPrefix = moduleName ? normalizePath(path.join("api", "proto", moduleName)) + "/" : null;

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

    // 1) собрать все файлы один раз
    const allFiles = await collectAllFiles(baseDir);

    // 2) найти сервисы по папкам cmd/*
    const services = await listCmdServices(baseDir);

    // 3) подготовить папку txt/ (очищаем)
    await ensureEmptyDir(outDir);

    // 4) migrations.txt
    {
        const matcher = makeMigrationsMatcher();
        const files = allFiles.filter((f) => matcher(f.relPath));
        const body = formatBundle("migrations", files);
        await fs.promises.writeFile(path.join(outDir, "migrations.txt"), body, "utf8");
    }

    // 5) каждый сервис -> свой txt
    // 5) каждый сервис -> свой txt
    for (const svc of services) {
        const matcher = makeServiceMatcher(baseDir, svc);
        const files = allFiles.filter((f) => matcher(f.relPath));
        const body = formatBundle(svc, files);
        await fs.promises.writeFile(path.join(outDir, `${svc}.txt`), body, "utf8");
    }

    // 6) index файл
    const indexLines = [];
    indexLines.push("=== generated txt bundles ===");
    indexLines.push(`baseDir: ${normalizePath(baseDir)}`);
    indexLines.push(`outDir: ${normalizePath(outDir)}`);
    indexLines.push("");
    indexLines.push(`services (${services.length}):`);
    for (const svc of services) indexLines.push(`- ${svc} -> ${svc}.txt`);
    indexLines.push("");
    indexLines.push("- migrations -> migrations.txt");

    await fs.promises.writeFile(path.join(outDir, "_index.txt"), indexLines.join("\n") + "\n", "utf8");

    console.log("Готово! Сгенерировано в папке:", outDir);
})();
