<#
.SYNOPSIS
Скрипт удаления службы Venera.
#>

$ErrorActionPreference = 'Stop'

# Требуются права администратора
$IsAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $IsAdmin) {
    Write-Warning "Для удаления службы требуются права администратора. Перезапустите скрипт от имени администратора."
    Exit
}

Write-Host "Удаление службы VeneraSrv..." -ForegroundColor Cyan

$svc = Get-Service -Name "VeneraSrv" -ErrorAction SilentlyContinue
if ($svc) {
    if ($svc.Status -eq 'Running') {
        Write-Host "Остановка службы..."
        Stop-Service -Name "VeneraSrv"
    }
    
    # Использование sc.exe для удаления
    & sc.exe delete VeneraSrv
    Write-Host "Служба успешно удалена." -ForegroundColor Green
} else {
    Write-Warning "Служба VeneraSrv не найдена."
}
