# Скрипт диагностики (п.9.3 ТЗ)
# Запускает Venera.exe с флагом --diagnose

$exePath = Join-Path $PSScriptRoot "..\Venera.exe"

if (-Not (Test-Path $exePath)) {
    Write-Host "Venera.exe не найден по пути: $exePath" -ForegroundColor Red
    Write-Host "Пожалуйста, сначала скомпилируйте проект (go build)."
    Exit 1
}

Write-Host "Запуск модуля диагностики Venera..." -ForegroundColor Cyan
& $exePath --diagnose

if ($LASTEXITCODE -eq 0) {
    Write-Host "Диагностика успешно завершена." -ForegroundColor Green
} else {
    Write-Host "Диагностика завершилась с ошибкой (Код: $LASTEXITCODE)." -ForegroundColor Red
}

Read-Host "Нажмите Enter для выхода..."
