<#
.SYNOPSIS
Скрипт-обертка для удаления контейнера СУБД DragonflyDB (п.15.4 ТЗ).
Вызывает бинарный файл venera.exe с параметром --remove_cachedb
#>

$ErrorActionPreference = 'Stop'

$ExePath = Join-Path -Path $PSScriptRoot -ChildPath "..\venera.exe"

if (-not (Test-Path $ExePath)) {
    Write-Error "Файл venera.exe не найден. Выполните сборку проекта."
    Exit
}

Write-Host "Удаление контейнера СУБД DragonflyDB..." -ForegroundColor Cyan

# Вызов команды --remove_cachedb (или -r)
& $ExePath --remove_cachedb

if ($LASTEXITCODE -eq 0) {
    Write-Host "Контейнер успешно удален." -ForegroundColor Green
} else {
    Write-Error "Возникла ошибка при выполнении операции (Код: $LASTEXITCODE)."
}
