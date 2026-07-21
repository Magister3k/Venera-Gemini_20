<#
.SYNOPSIS
Скрипт сборки проекта Venera для PowerShell.
#>

$ErrorActionPreference = 'Stop'

Write-Host "Начинаем сборку проекта Venera..." -ForegroundColor Cyan

# 1. Проверка наличия Go
try {
    go version | Out-Null
} catch {
    Write-Error "Go не установлен или не найден в PATH."
}

# 2. Опциональная сборка React-UI (если установлен Node.js и npm)
try {
    npm -v | Out-Null
    Write-Host "Сборка React-UI (SPA)..." -ForegroundColor Cyan
    Push-Location "react-ui"
    npm install
    npm run build
    Pop-Location
} catch {
    Write-Warning "npm не найден. Сборка React-UI пропущена. Пожалуйста, установите Node.js."
}

# 3. Загрузка зависимостей
Write-Host "Загрузка зависимостей..." -ForegroundColor Cyan
go mod tidy

# 4. Внедрение иконки в тело программы
# Для интеграции иконки используется go-winres, так как это стандарт де-факто в Go
try {
    go-winres version | Out-Null
    Write-Host "Генерация ресурсов (внедрение иконки из assets/image.ico)..." -ForegroundColor Cyan
    # Если go-winres установлен, он соберет rsrc.syso
    go-winres make
} catch {
    Write-Warning "Утилита go-winres не найдена. Установите 'go install github.com/tc-hib/go-winres@latest' для внедрения иконки."
}

# 5. Сборка исполняемого файла
Write-Host "Компиляция..." -ForegroundColor Cyan
# Оптимизация сборки (уменьшение размера: -s -w)
go build -ldflags="-s -w" -o venera.exe main.go

if ($LASTEXITCODE -eq 0) {
    Write-Host "Сборка успешно завершена. Исполняемый файл: venera.exe" -ForegroundColor Green
} else {
    Write-Error "Ошибка при компиляции."
}

