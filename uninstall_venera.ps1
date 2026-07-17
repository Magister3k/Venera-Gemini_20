<#
.SYNOPSIS
Скрипт удаления службы Venera.
Этот скрипт использует встроенный механизм приложения (п.15.7 ТЗ).
#>

$ErrorActionPreference = 'Stop'

# Требуются права администратора (п.16.2 ТЗ)
$IsAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $IsAdmin) {
    Write-Warning "Для удаления службы требуются права администратора. Перезапустите скрипт от имени администратора."
    Exit
}

$ExePath = Join-Path -Path $PWD -ChildPath "venera.exe"
if (-not (Test-Path $ExePath)) {
    Write-Error "Файл venera.exe не найден. Возможно, приложение уже удалено."
    Exit
}

Write-Host "Удаление службы VeneraSrv через внутренний модуль приложения..." -ForegroundColor Cyan

# Используем флаг --uninstall_srv (или -u) из п.15.7 ТЗ
& $ExePath --uninstall_srv

if ($LASTEXITCODE -eq 0) {
    Write-Host "Служба успешно удалена." -ForegroundColor Green
} else {
    Write-Warning "Возникла ошибка при удалении службы (Код: $LASTEXITCODE). Возможно, служба не была установлена."
}

