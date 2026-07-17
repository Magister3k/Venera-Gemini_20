; Скрипт создания инсталлятора для Inno Setup (http://www.jrsoftware.org/isinfo.php)
#define MyAppName "Venera Collector"
#define MyAppVersion "1.0.0"
#define MyAppPublisher "Venera Team"
#define MyAppExeName "venera.exe"

[Setup]
; Уникальный идентификатор приложения. Не меняйте его для обновлений!
AppId={{B16B47AB-3286-4F7F-BB8E-1768C8BCF6FC}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
DefaultDirName={autopf}\Venera
DefaultGroupName={#MyAppName}
OutputBaseFilename=Venera_Setup_v{#MyAppVersion}
Compression=lzma2/ultra
SolidCompression=yes
; Для установки службы требуются права администратора
PrivilegesRequired=admin
ArchitecturesInstallIn64BitMode=x64
SetupIconFile=assets\image.ico
UninstallDisplayIcon={app}\{#MyAppExeName}

[Languages]
Name: "russian"; MessagesFile: "compiler:Languages\Russian.isl"
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked

[Files]
; Основной исполняемый файл
Source: "venera.exe"; DestDir: "{app}"; Flags: ignoreversion

; Файлы конфигурации (копируем только если их еще нет, чтобы не затереть настройки при обновлении)
Source: "config.toml"; DestDir: "{app}"; Flags: onlyifdoesntexist
Source: "processes.toml"; DestDir: "{app}"; Flags: onlyifdoesntexist
Source: "manifest.xml"; DestDir: "{app}"; Flags: ignoreversion

; Директории
Source: "react-ui\dst\*"; DestDir: "{app}\react-ui\dst"; Flags: ignoreversion recursesubdirs createallsubdirs
Source: "scripts\*"; DestDir: "{app}\scripts"; Flags: ignoreversion recursesubdirs createallsubdirs
Source: "sql\*"; DestDir: "{app}\sql"; Flags: ignoreversion recursesubdirs createallsubdirs
Source: "assets\*"; DestDir: "{app}\assets"; Flags: ignoreversion recursesubdirs createallsubdirs

; Настройки фильтров и алертов (если файлов нет - копируются заглушки)
Source: "settings\*"; DestDir: "{app}\settings"; Flags: onlyifdoesntexist recursesubdirs createallsubdirs

[Dirs]
; Создаем пустые директории для работы
Name: "{app}\Logs"
Name: "{app}\backups"

[Icons]
Name: "{group}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"
Name: "{group}\Диагностика Venera"; Filename: "powershell.exe"; Parameters: "-ExecutionPolicy Bypass -WindowStyle Hidden -File ""{app}\scripts\diagnose-gui.ps1"""
Name: "{group}\{cm:UninstallProgram,{#MyAppName}}"; Filename: "{uninstallexe}"
Name: "{commondesktop}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; Tasks: desktopicon

[Run]
; Автоматическая установка службы при завершении установки
Filename: "{app}\{#MyAppExeName}"; Parameters: "--install_srv"; Flags: runhidden; StatusMsg: "Установка службы VeneraSrv..."

[UninstallRun]
; Автоматическое удаление службы при деинсталляции
Filename: "{app}\{#MyAppExeName}"; Parameters: "--uninstall_srv"; Flags: runhidden; RunOnceId: "RemoveVeneraService"
