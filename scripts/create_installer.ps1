<#
.SYNOPSIS
Скрипт для сборки и упаковки приложения в инсталляционный пакет (.exe) или Portable архив (.zip).
#>

$ErrorActionPreference = 'Stop'

$ProjectRoot = Join-Path $PSScriptRoot ".."
Push-Location $ProjectRoot

Write-Host "=== Подготовка к созданию инсталляционного пакета Venera ===" -ForegroundColor Cyan

# 1. Запуск основной сборки приложения
Write-Host "Шаг 1: Компиляция бинарных файлов и React-UI..." -ForegroundColor Yellow
& .\build.ps1
if ($LASTEXITCODE -ne 0) {
    Write-Error "Ошибка сборки проекта. Создание пакета отменено."
    Pop-Location
    Exit 1
}

# Проверяем, существует ли папка settings. Если нет - создадим шаблоны, чтобы инсталлятору было что копировать
if (-Not (Test-Path "settings")) {
    New-Item -ItemType Directory -Path "settings" | Out-Null
    Set-Content -Path "settings\generic.flt" -Value ""
    Set-Content -Path "settings\generic.ctr" -Value ""
    Set-Content -Path "settings\generic.alr" -Value ""
}

# 2. Поиск компилятора Inno Setup
$ISCC = "${env:ProgramFiles(x86)}\Inno Setup 6\ISCC.exe"
if (-not (Test-Path $ISCC)) {
    $ISCC = "$env:ProgramFiles\Inno Setup 6\ISCC.exe"
}

if (Test-Path $ISCC) {
    Write-Host "Шаг 2: Создание .exe инсталлятора через Inno Setup..." -ForegroundColor Yellow
    $IssPath = Join-Path $ProjectRoot "installer.iss"
    
    # Запускаем компилятор Inno Setup
    & $ISCC $IssPath
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Установщик успешно создан в папке Output!" -ForegroundColor Green
    } else {
        Write-Error "Ошибка при генерации установщика (Код: $LASTEXITCODE)."
    }
} else {
    Write-Warning "Inno Setup 6 не найден. Пропуск создания .exe установщика."
    Write-Host "Переход к созданию переносимого Portable (.zip) архива..." -ForegroundColor Yellow
    
    $ReleaseDir = "Venera_Release"
    if (Test-Path $ReleaseDir) { Remove-Item -Recurse -Force $ReleaseDir }
    New-Item -ItemType Directory -Path $ReleaseDir | Out-Null
    
    # Копирование файлов в релизную папку
    Copy-Item "venera.exe" -Destination $ReleaseDir
    Copy-Item "config.toml" -Destination $ReleaseDir
    Copy-Item "processes.toml" -Destination $ReleaseDir
    Copy-Item "manifest.xml" -Destination $ReleaseDir
    Copy-Item "react-ui\dst" -Destination "$ReleaseDir\react-ui" -Recurse
    Copy-Item "scripts" -Destination $ReleaseDir -Recurse
    Copy-Item "sql" -Destination $ReleaseDir -Recurse
    Copy-Item "assets" -Destination $ReleaseDir -Recurse
    Copy-Item "settings" -Destination $ReleaseDir -Recurse
    
    New-Item -ItemType Directory -Path "$ReleaseDir\Logs" | Out-Null
    New-Item -ItemType Directory -Path "$ReleaseDir\backups" | Out-Null
    
    $ZipPath = "Venera_Portable.zip"
    if (Test-Path $ZipPath) { Remove-Item -Force $ZipPath }
    
    Compress-Archive -Path "$ReleaseDir\*" -DestinationPath $ZipPath
    Remove-Item -Recurse -Force $ReleaseDir
    
    Write-Host "Создан переносимый архив: $ZipPath" -ForegroundColor Green
}

Pop-Location
Write-Host "=== Процесс упаковки завершен ===" -ForegroundColor Cyan
