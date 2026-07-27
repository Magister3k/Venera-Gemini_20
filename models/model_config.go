package models

import "time"

// Config структура представляет конфигурацию приложения, загружаемую из файла config.toml.
type Config struct {
	Generic     GenericConfig     `toml:"Generic"`     // Общие настройки
	Paths       PathsConfig       `toml:"Paths"`       // Пути к зависимостям и файлам
	DragonflyDB DragonflyDBConfig `toml:"DragonflyDB"` // Настройки кэширующей СУБД (Redis/DragonflyDB)
	PostgreSQL  PostgreSQLConfig  `toml:"PostgreSQL"`  // Настройки СУБД PostgreSQL
	System      SystemConfig      `toml:"System"`      // Системные пороги для защиты
}

// GenericConfig содержит общие настройки приложения.
type GenericConfig struct {
	Mode                 string `toml:"mode"`                    // Режим работы: "tray" (системный трей) или "service" (служба)
	AutoStart            bool   `toml:"auto_start"`              // Автоматический старт всех процессов при запуске
	MaxProcesses         int    `toml:"max_processes"`           // Максимальное количество одновременных процессов обработки
	WebServerPort        int    `toml:"web_server_port"`         // Порт для веб-интерфейса
	LogRotationDays      int    `toml:"log_rotation_days"`       // Количество дней для хранения логов до их ротации
	LocalUIType          string `toml:"local_ui_type"`           // Тип локального интерфейса: "static" или "react"
	ShowConsoleOnStartup bool   `toml:"show_console_on_startup"` // true - показывать консоль при старте, false - сразу скрывать
}

// PathsConfig содержит пути к внешним зависимостям и конфигурационным файлам.
type PathsConfig struct {
	PodmanExe   string `toml:"podman_exe"`    // Путь к исполняемому файлу Podman
	TsharkExe   string `toml:"tshark_exe"`    // Путь к исполняемому файлу Tshark
	FilterList  string `toml:"filter_list"`   // Путь к файлу со списком фильтрации данных generic.flt
	ControlList string `toml:"control_list"`  // Путь к файлу со списоком значений на контроле generic.ctr
	AlertsList  string `toml:"alerts_list"`   // Путь к файлу с правилами формирования алертов generic.alr
	DbImage     string `toml:"db_image"`      // Путь к локальному образу кэширующей СУБД
	DbBackupDir string `toml:"db_backup_dir"` // Путь к директории для размещения резервных копий кэширующей СУБД
}

// DragonflyDBConfig содержит параметры подключения и работы с кэширующей СУБД.
type DragonflyDBConfig struct {
	Host      string        `toml:"host"`       // IP или хост подключения
	Port      int           `toml:"port"`       // Порт подключения
	Password  string        `toml:"password"`   // Пароль (если установлен)
	BatchSize int           `toml:"batch_size"` // Количество записей для пакетной обработки в list
	Timeout   time.Duration `toml:"timeout"`    // Таймер ожидания для пакетной обработки записей в list, если порог не достигнут
}

// PostgreSQLConfig содержит параметры подключения к СУБД PostgreSQL.
type PostgreSQLConfig struct {
	Host     string `toml:"host"`     // IP или хост подключения
	Port     int    `toml:"port"`     // Порт подключения
	User     string `toml:"user"`     // Имя пользователя
	Password string `toml:"password"` // Пароль
	Database string `toml:"database"` // Имя БД
	SSLMode  string `toml:"ssl_mode"` // Режим SSL (disable/require)
}

// SystemConfig содержит параметры системных ограничений для защиты от переполнения.
type SystemConfig struct {
	DiskWarningThreshold  float64 `toml:"disk_warning_threshold"`  // Процент свободного места на диске, при котором выдавать предупреждение
	DiskCriticalThreshold float64 `toml:"disk_critical_threshold"` // Процент свободного места на диске, при котором останавливать все процессы
	RamCriticalThreshold  float64 `toml:"ram_critical_threshold"`  // Процент свободной RAM, при котором останавливать все процессы
}
