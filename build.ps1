<#
.SYNOPSIS
Скрипт сборки проекта Venera.
#>

$ErrorActionPreference = 'Stop'

Write-Host "Начинаем сборку проекта Venera..." -ForegroundColor Cyan

# Проверка наличия Go
try {
    go version | Out-Null
} catch {
    Write-Error "Go не установлен или не найден в PATH."
}

# Загрузка зависимостей
Write-Host "Загрузка зависимостей..." -ForegroundColor Cyan
go mod tidy

# Сборка исполняемого файла
Write-Host "Компиляция..." -ForegroundColor Cyan
go build -o venera.exe

if ($LASTEXITCODE -eq 0) {
    Write-Host "Сборка успешно завершена. Исполняемый файл: venera.exe" -ForegroundColor Green
} else {
    Write-Error "Ошибка при компиляции."
}
