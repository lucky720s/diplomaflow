const fs = require("fs");
const path = require("path");

async function collectFiles(dir) {
    const results = [];

    async function walk(currentPath) {
        const items = await fs.promises.readdir(currentPath, { withFileTypes: true });

        for (const item of items) {
            if (item.name.startsWith(".")) continue;

            const fullPath = path.join(currentPath, item.name);

            let stat;
            try {
                stat = await fs.promises.stat(fullPath);
            } catch {
                continue;
            }

            if (stat.isDirectory()) {
                await walk(fullPath);
            } else if (stat.isFile()) {
                if (
                    item.name.endsWith(".pb.go") ||
                    item.name.endsWith(".js") ||
                    item.name.endsWith(".txt") ||
                    item.name.endsWith("validate.go") ||
                    item.name.endsWith(".mod") ||
                    item.name.endsWith(".work") ||
                    item.name.endsWith(".sum") ||
                    item.name.endsWith(".exe")
                ) continue;

                const content = await fs.promises.readFile(fullPath, "utf8");

                results.push({
                    path: fullPath,
                    content
                });
            }
        }
    }

    await walk(dir);
    return results;
}

async function readEnvFile(envPath) {
    try {
        const content = await fs.promises.readFile(envPath, "utf8");
        return content;
    } catch (err) {
        console.error("Ошибка при чтении .env файла:", err);
        return null;
    }
}

function normalizePath(filePath) {
    return filePath.split(path.sep).join('/');
}

(async () => {
    const startDir = path.resolve("./");
    const baseDir = path.resolve("./");
   // const baseDir = path.resolve("internal/workflow");
    const files = await collectFiles(baseDir);

    let output = "=== Собранные файлы ===\n\n";

    const envPath = path.join(baseDir, ".env");
    const envContent = await readEnvFile(envPath);

    if (envContent) {
        output += "=== .env Файл ===\n";
        output += `${envContent}\n`;
        output += "----------------------------------------\n\n";
    }

    for (const file of files) {
        output += `===== FILE START =====\n`;
        output += `PATH: ${normalizePath(file.path)}\n`;
        output += `===== FILE CONTENT =====\n`;
        output += `${file.content}\n`;
        output += `===== FILE END =====\n\n`;
        output += `SIZE: ${Buffer.byteLength(file.content, "utf8")} bytes\n`;

    }


    output = output
        .split("\n")
        .filter(line => line.trim() !== "")
        .join("\n");

    const txtPath = path.join(__dirname, "collected.txt");

    if (fs.existsSync(txtPath)) {
        fs.unlinkSync(txtPath);
    }

    fs.writeFileSync(txtPath, output, "utf8");

    console.log("Готово! Файл сохранён:");
    console.log(txtPath);
})();
