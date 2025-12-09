const fs = require("fs");
const path = require("path");

async function collectFiles(dir) {
    const results = [];

    async function walk(currentPath) {
        const items = await fs.promises.readdir(currentPath, { withFileTypes: true });

        for (const item of items) {
            if (item.name.startsWith(".")) continue;

            const fullPath = path.join(currentPath, item.name);

            if (item.isDirectory()) {
                await walk(fullPath);

            } else if (item.isFile()) {
                // --- ПРОПУСКАЕМ ВСЕ .pb.go ФАЙЛЫ ---
                if (item.name.endsWith(".pb.go")) continue;
                if (item.name.endsWith(".js")) continue;
                if (item.name.endsWith(".txt")) continue;
                if (item.name.endsWith("validate.go")) continue;
                if (item.name.endsWith(".mod")) continue;
                if (item.name.endsWith(".work")) continue;
                if (item.name.endsWith(".sum")) continue;

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

// === Запуск ===
(async () => {
    const startDir = path.resolve("./");
    const files = await collectFiles(startDir);

    let output = "=== Собранные файлы ===\n\n";

    for (const file of files) {
        output += `FILE: ${file.path}\n`;
        output += `CONTENT:\n${file.content}\n`;
        output += "----------------------------------------\n\n";
    }
    output = output
        .split("\n")
        .filter(line => line.trim() !== "")
        .join("\n");
    const txtPath = path.join(__dirname, "collected.txt");

    // --- УДАЛЯЕМ ФАЙЛ collected.txt ПЕРЕД ЗАПУСКОМ ---
    if (fs.existsSync(txtPath)) {
        fs.unlinkSync(txtPath);
    }

    fs.writeFileSync(txtPath, output, "utf8");

    console.log("Готово! Файл сохранён:");
    console.log(txtPath);
})();
