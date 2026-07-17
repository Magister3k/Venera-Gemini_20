<#
.SYNOPSIS
Скрипт-обертка для создания контейнера СУБД DragonflyDB (п.15.3 ТЗ).
Вызывает бинарный файл venera.exe с параметром --create_cachedb
#>

$ErrorActionPreference = 'Stop'

$ExePath = Join-Path -Path $PSScriptRoot -ChildPath "..\venera.exe"

if (-not (Test-Path $ExePath)) {
    Write-Error "Файл venera.exe не найден. Выполните сборку проекта."
    Exit
}

Write-Host "Запуск создания контейнера СУБД DragonflyDB..." -ForegroundColor Cyan

# Вызов команды --create_cachedb (или -c)
& $ExePath --create_cachedb

if ($LASTEXITCODE -eq 0) {
    Write-Host "Операция успешно завершена." -ForegroundColor Green
} else {
    Write-Error "Возникла ошибка при выполнении операции (Код: $LASTEXITCODE)."
}
