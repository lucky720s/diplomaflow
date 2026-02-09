const fs = require('fs');
const path = require('path');

// --- НАСТРОЙКИ ---
const SOURCE_DIR = './';        // Где искать файлы (текущая папка)
const OUTPUT_FILE = 'admin_code_bundle.txt'; // Куда сохранять
const FILTER = 'admin';         // Ищем файлы, где в названии или пути есть "admin"
const EXTENSION = '.go';        // Нас интересуют только файлы Go
// -----------------

function getFiles(dir, allFiles = []) {
    const files = fs.readdirSync(dir);

    files.forEach(file => {
        const name = path.join(dir, file);
        // Игнорируем папку node_modules и скрытые папки (напр. .git)
        if (file === 'node_modules' || file.startsWith('.')) return;

        if (fs.statSync(name).isDirectory()) {
            getFiles(name, allFiles);
        } else {
            // Проверяем расширение и наличие фильтра в пути
            if (name.endsWith(EXTENSION) && name.toLowerCase().includes(FILTER)) {
                allFiles.push(name);
            }
        }
    });
    return allFiles;
}

const filesToCopy = getFiles(SOURCE_DIR);
let finalContent = '';

filesToCopy.forEach(filePath => {
    const content = fs.readFileSync(filePath, 'utf8');
    finalContent += `\n\n// --- FILE: ${filePath} ---\n\n`;
    finalContent += content;
});

fs.writeFileSync(OUTPUT_FILE, finalContent);
console.log(`Готово! Собрано файлов: ${filesToCopy.length}. Результат в ${OUTPUT_FILE}`);