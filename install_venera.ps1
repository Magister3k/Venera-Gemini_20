<#
.SYNOPSIS
Скрипт установки Venera как службы Windows.
Этот скрипт использует встроенный механизм приложения.
#>

$ErrorActionPreference = 'Stop'

# Требуются права Администратора
$IsAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $IsAdmin) {
    Write-Warning "Для установки службы требуются права Администратора. Перезапустите скрипт от имени администратора."
    Exit
}

$ExePath = Join-Path -Path $PWD -ChildPath "venera.exe"
if (-not (Test-Path $ExePath)) {
    Write-Error "Файл venera.exe не найден. Выполните сборку проекта."
    Exit
}

Write-Host "Установка службы VeneraSrv через внутренний модуль приложения..." -ForegroundColor Cyan

# Используем флаг --install_srv (или -i)
& $ExePath --install_srv

if ($LASTEXITCODE -eq 0) {
    Write-Host "Операция успешно завершена." -ForegroundColor Green
    
    # Попытка автоматического запуска
    Write-Host "Попытка запуска службы..."
    Start-Service -Name "VeneraSrv" -ErrorAction SilentlyContinue
    if ($?) {
        Write-Host "Служба VeneraSrv запущена." -ForegroundColor Green
    } else {
        Write-Warning "Не удалось запустить службу. Проверьте конфигурацию."
    }
} else {
    Write-Error "Возникла ошибка при установке службы (Код: $LASTEXITCODE)."
}

