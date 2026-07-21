<#
.SYNOPSIS
Скрипт удаления службы Venera.
Этот скрипт использует встроенный механизм приложения.
#>

$ErrorActionPreference = 'Stop'

# Требуются права Администратора
$IsAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $IsAdmin) {
    Write-Warning "Для удаления службы требуются права Администратора. Перезапустите скрипт от имени Администратора."
    Exit
}

$ExePath = Join-Path -Path $PWD -ChildPath "venera.exe"
if (-not (Test-Path $ExePath)) {
    Write-Error "Файл venera.exe не найден. Возможно, приложение уже удалено."
    Exit
}

Write-Host "Удаление службы VeneraSrv через внутренний модуль приложения..." -ForegroundColor Cyan

# Используем флаг --uninstall_srv (или -u)
& $ExePath --uninstall_srv

if ($LASTEXITCODE -eq 0) {
    Write-Host "Служба успешно удалена." -ForegroundColor Green
} else {
    Write-Warning "Возникла ошибка при удалении службы (Код: $LASTEXITCODE). Возможно, служба не была установлена."
}

