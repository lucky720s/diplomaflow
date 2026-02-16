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
                    item.name.endsWith(".pub") ||
                    item.name.endsWith("loy") ||
                    item.name.endsWith(".exe")
                ) continue;

                const content = await fs.promises.readFile(fullPath, "utf8");

                results.push({
                    path: fullPath,
                    content,
                    size: Buffer.byteLength(content, "utf8")
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
    } catch {
        return null;
    }
}

function normalizePath(filePath) {
    return filePath.split(path.sep).join('/');
}

function buildOutput(header, envContent, files) {
    let output = header + "\n\n";

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
        output += `SIZE: ${file.size} bytes\n\n`;
    }

    return output;
}

(async () => {
    const baseDir = path.resolve("./");
    const files = await collectFiles(baseDir);

    const envPath = path.join(baseDir, ".env");
    const envContent = await readEnvFile(envPath);

    // Считаем общий размер
    const totalSize = files.reduce((sum, f) => sum + f.size, 0);
    const halfSize = totalSize / 2;

    const part1 = [];
    const part2 = [];

    let currentSize = 0;

    for (const file of files) {
        if (currentSize < halfSize) {
            part1.push(file);
            currentSize += file.size;
        } else {
            part2.push(file);
        }
    }

    const output1 = buildOutput("=== Собранные файлы (PART 1) ===", envContent, part1);
    const output2 = buildOutput("=== Собранные файлы (PART 2) ===", null, part2);

    const txtPath1 = path.join(__dirname, "collected1.txt");
    const txtPath2 = path.join(__dirname, "collected2.txt");

    if (fs.existsSync(txtPath1)) fs.unlinkSync(txtPath1);
    if (fs.existsSync(txtPath2)) fs.unlinkSync(txtPath2);

    fs.writeFileSync(txtPath1, output1, "utf8");
    fs.writeFileSync(txtPath2, output2, "utf8");

    console.log("Готово!");
    console.log("Файл 1:", txtPath1);
    console.log("Файл 2:", txtPath2);
})();
