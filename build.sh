#!/bin/bash
# Скрипт сборки проекта Venera для среды bash (п.25 ТЗ)
# Использование: ./build.sh

set -e

echo -e "\033[0;36mНачинаем сборку проекта Venera...\033[0m"

# 1. Проверка наличия Go
if ! command -v go &> /dev/null
then
    echo -e "\033[0;31mGo не установлен или не найден в PATH.\033[0m"
    exit 1
fi

echo "Версия Go:"
go version

# 2. Опциональная сборка React-UI (если установлен Node.js и npm)
if command -v npm &> /dev/null
then
    echo -e "\033[0;36mСборка React-UI (SPA)...\033[0m"
    cd react-ui
    npm install
    npm run build
    cd ..
else
    echo -e "\033[0;33mВНИМАНИЕ: npm не найден. Сборка React-UI пропущена.\033[0m"
    echo -e "Пожалуйста, установите Node.js для сборки веб-интерфейса, или используйте предварительно скомпилированные файлы."
fi

# 3. Загрузка зависимостей
echo -e "\033[0;36mЗагрузка зависимостей Go...\033[0m"
go mod tidy

# 4. Внедрение иконки (п.21 ТЗ) - опционально для Windows сборок на Linux
if command -v go-winres &> /dev/null
then
    echo -e "\033[0;36mГенерация ресурсов (Иконка Windows)...\033[0m"
    go-winres make
fi

# 5. Сборка исполняемого файла
echo -e "\033[0;36mКомпиляция Go проекта...\033[0m"
# Для сборки Windows-экзешника из Linux
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o venera.exe main.go

if [ $? -eq 0 ]; then
    echo -e "\033[0;32mСборка успешно завершена. Исполняемый файл: venera.exe\033[0m"
else
    echo -e "\033[0;31mОшибка при компиляции.\033[0m"
    exit 1
fi
