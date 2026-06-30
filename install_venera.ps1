<#
.SYNOPSIS
Скрипт установки Venera как службы Windows.
#>

$ErrorActionPreference = 'Stop'

# Требуются права администратора
$IsAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $IsAdmin) {
    Write-Warning "Для установки службы требуются права администратора. Перезапустите скрипт от имени администратора."
    Exit
}

$ExePath = Join-Path -Path $PWD -ChildPath "venera.exe"
if (-not (Test-Path $ExePath)) {
    Write-Error "Файл venera.exe не найден. Выполните сборку."
    Exit
}

Write-Host "Установка службы VeneraSrv..." -ForegroundColor Cyan

# Проверка, существует ли уже служба
$svc = Get-Service -Name "VeneraSrv" -ErrorAction SilentlyContinue
if ($svc) {
    Write-Warning "Служба VeneraSrv уже установлена."
} else {
    New-Service -Name "VeneraSrv" -BinaryPathName $ExePath -DisplayName "Venera Service" -Description "Система сбора идентификаторов Venera" -StartupType Automatic
    Write-Host "Служба успешно установлена." -ForegroundColor Green
}

# Попытка запуска
try {
    Start-Service -Name "VeneraSrv"
    Write-Host "Служба VeneraSrv запущена." -ForegroundColor Green
} catch {
    Write-Warning "Не удалось запустить службу. Проверьте логи."
}
