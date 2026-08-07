package models

import "time"

// Config представляет конфигурацию приложения, загружаемую из файла config.toml.
type Config struct {
	Generic     GenericCfg     `toml:"Generic"`     // Общие настройки приложения
	Paths       PathsCfg       `toml:"Paths"`       // Пути к внешним зависимостям и файлам настроек
	CacheDb     CacheDbCfg     `toml:"CacheDb"`     // Параметры подключения и работы с кэширующей СУБД
	PgDb        PgDbCfg        `toml:"PostgreSQL"`  // Параметры подключения к итоговой базе в СУБД PostgreSQL
	System      SysCfg         `toml:"System"`      // Параметры системных ограничений для защиты от переполнения
}

// GenericCfg содержит общие настройки приложения.
type GenericCfg struct {
	Mode                 string `toml:"mode"`                    // Режим работы: "tray" - системный трей, "service" - служба Windows
	AutoStart            bool   `toml:"auto_start"`              // Автоматический старт всех процессов при запуске
	MaxProcs             int    `toml:"max_processes"`           // Максимальное количество одновременных процессов обработки
	WebSrvPort           int    `toml:"web_server_port"`         // Порт для веб-интерфейса
	LogRotDays           int    `toml:"log_rotation_days"`       // Количество дней для хранения логов до их ротации
	LocalUIType          string `toml:"local_ui_type"`           // Тип локального веб-интерфейса: "static" - REST API, "react" - React UI
	ShowConsoleOnStartup bool   `toml:"show_console_on_startup"` // Консоль при старте: "true" - показывать, "false" - скрывать
}

// PathsCfg содержит пути к внешним зависимостям и файлам настроек.
type PathsCfg struct {
	Podman       string `toml:"podman_exe"`    // Путь к исполняемому файлу Podman
	Tshark       string `toml:"tshark_exe"`    // Путь к исполняемому файлу Tshark
	Filter       string `toml:"filter_list"`   // Путь к файлу со списком фильтрации данных generic.flt
	Control      string `toml:"control_list"`  // Путь к файлу со списоком значений на контроле generic.ctr
	Alerts       string `toml:"alerts_list"`   // Путь к файлу с правилами формирования алертов generic.alr
	CacheDbImage string `toml:"cachedb_image"` // Путь к локальному образу кэширующей СУБД
	CacheDir     string `toml:"chache_dir"`    // Путь к директории для размещения резервных данных кэширующей СУБД
}

// CacheDbCfg содержит параметры подключения и работы с кэширующей СУБД.
type CacheDbCfg struct {
	Host      string        `toml:"host"`       // IP или хост подключения
	Port      int           `toml:"port"`       // Порт подключения
	Pass      string        `toml:"password"`   // Пароль (если установлен)
	BatchSize int           `toml:"batch_size"` // Количество записей для пакетной обработки в list
	Timeout   time.Duration `toml:"timeout"`    // Таймер ожидания для пакетной обработки записей в list, если порог не достигнут
}

// PgDbCfg содержит параметры подключения к итоговой базе в СУБД PostgreSQL.
type PgDbCfg struct {
	Host     string `toml:"host"`     // IP или хост подключения
	Port     int    `toml:"port"`     // Порт подключения
	User     string `toml:"user"`     // Имя пользователя
	Pass     string `toml:"password"` // Пароль
	Name     string `toml:"name"`     // Имя БД
	SSLMode  string `toml:"ssl_mode"` // Режим SSL (disable/require)
}

// SysCfg содержит параметры системных ограничений для защиты от переполнения.
type SysCfg struct {
	DiskWarningThreshold  float64 `toml:"disk_warning_threshold"`  // Процент свободного места на диске, при котором выдавать предупреждение
	DiskCriticalThreshold float64 `toml:"disk_critical_threshold"` // Процент свободного места на диске, при котором останавливать все процессы
	RamCriticalThreshold  float64 `toml:"ram_critical_threshold"`  // Процент свободной RAM, при котором останавливать все процессы
}
