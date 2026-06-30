package models

import "time"

// Config структура представляет конфигурацию приложения, загружаемую из config.toml.
type Config struct {
	Generic     GenericConfig     `toml:"Generic"`
	Paths       PathsConfig       `toml:"Paths"`
	DragonflyDB DragonflyDBConfig `toml:"DragonflyDB"`
	PostgreSQL  PostgreSQLConfig  `toml:"PostgreSQL"`
	System      SystemConfig      `toml:"System"`
}

// GenericConfig содержит общие настройки приложения.
type GenericConfig struct {
	Mode             string `toml:"mode"`             // Режим работы: "tray" или "service"
	AutoStart        bool   `toml:"auto_start"`       // Автоматический старт процессов
	MaxProcesses     int    `toml:"max_processes"`    // Максимальное количество одновременных процессов обработки
	WebServerPort    int    `toml:"web_server_port"`  // Порт для веб-интерфейса
	LogRotationDays  int    `toml:"log_rotation_days"` // Количество дней для хранения логов до ротации
}

// PathsConfig содержит пути к внешним зависимостям и файлам данных.
type PathsConfig struct {
	PodmanExe   string `toml:"podman_exe"`
	TsharkExe   string `toml:"tshark_exe"`
	FilterList  string `toml:"filter_list"`  // Путь к generic.flt (белый список ключей, черный список значений)
	ControlList string `toml:"control_list"` // Путь к generic.ctr (список значений на контроле)
	AlertsList  string `toml:"alerts_list"`  // Путь к generic.alr
	DbImage     string `toml:"db_image"`     // Образ СУБД DragonflyDB (например, docker.io/dragonflydb/dragonfly)
	DbBackupDir string `toml:"db_backup_dir"` // Папка для резервных копий
}

// DragonflyDBConfig параметры подключения и работы с in-memory СУБД.
type DragonflyDBConfig struct {
	Host        string        `toml:"host"`
	Port        int           `toml:"port"`
	Password    string        `toml:"password"`
	BatchSize   int           `toml:"batch_size"` // Количество записей для пакетной обработки (порог)
	Timeout     time.Duration `toml:"timeout"`    // Время ожидания перед принудительной обработкой (в секундах или мс)
}

// PostgreSQLConfig параметры подключения к целевой реляционной СУБД.
type PostgreSQLConfig struct {
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	User     string `toml:"user"`
	Password string `toml:"password"`
	Database string `toml:"database"`
	SSLMode  string `toml:"ssl_mode"`
}

// SystemConfig параметры системных ограничений для защиты от переполнения.
type SystemConfig struct {
	DiskWarningThreshold  float64 `toml:"disk_warning_threshold"` // Процент, при котором выдавать предупреждение
	DiskCriticalThreshold float64 `toml:"disk_critical_threshold"` // Процент, при котором останавливать процессы
	RamCriticalThreshold  float64 `toml:"ram_critical_threshold"`  // Процент свободной RAM для остановки
}
