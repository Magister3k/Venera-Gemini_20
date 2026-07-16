package models

import "time"

// Config структура представляет конфигурацию приложения, загружаемую из файла config.toml (п.1.11 ТЗ).
type Config struct {
	Generic     GenericConfig     `toml:"Generic"`     // Общие настройки
	Paths       PathsConfig       `toml:"Paths"`       // Пути к зависимостям и файлам
	DragonflyDB DragonflyDBConfig `toml:"DragonflyDB"` // Настройки in-memory СУБД
	PostgreSQL  PostgreSQLConfig  `toml:"PostgreSQL"`  // Настройки реляционной СУБД
	System      SystemConfig      `toml:"System"`      // Системные пороги для защиты
}

// GenericConfig содержит общие настройки приложения (раздел Generic по п.1.11 ТЗ).
type GenericConfig struct {
	Mode             string `toml:"mode"`              // Режим работы: "tray" (системный трей) или "service" (служба)
	AutoStart        bool   `toml:"auto_start"`        // Автоматический старт процессов при запуске
	MaxProcesses     int    `toml:"max_processes"`     // Максимальное количество одновременных процессов обработки (п.1.5 ТЗ: до 20)
	WebServerPort    int    `toml:"web_server_port"`   // Порт для веб-интерфейса
	LogRotationDays  int    `toml:"log_rotation_days"` // Количество дней для хранения логов до их ротации (п.7.1 ТЗ)
}

// PathsConfig содержит пути к внешним зависимостям и конфигурационным файлам (раздел Paths по п.1.11 ТЗ).
type PathsConfig struct {
	PodmanExe   string `toml:"podman_exe"`    // Путь к исполняемому файлу Podman (п.2.7)
	TsharkExe   string `toml:"tshark_exe"`    // Путь к исполняемому файлу Tshark (п.2.14)
	FilterList  string `toml:"filter_list"`   // Путь к файлу generic.flt: списки фильтрации данных (п.1.7 ТЗ)
	ControlList string `toml:"control_list"`  // Путь к файлу generic.ctr: список значений на контроле (п.1.8 ТЗ)
	AlertsList  string `toml:"alerts_list"`   // Путь к файлу generic.alr: правила формирования алертов (п.1.10 ТЗ)
	DbImage     string `toml:"db_image"`      // Путь к локальному образу СУБД DragonflyDB (п.2.9 ТЗ)
	DbBackupDir string `toml:"db_backup_dir"` // Папка для размещения резервных копий СУБД DragonflyDB
}

// DragonflyDBConfig содержит параметры подключения и работы с СУБД DragonflyDB (раздел DragonflyDB по п.1.11 ТЗ).
type DragonflyDBConfig struct {
	Host        string        `toml:"host"`       // IP или хост подключения
	Port        int           `toml:"port"`       // Порт подключения (обычно 6379)
	Password    string        `toml:"password"`   // Пароль для доступа (если установлен)
	BatchSize   int           `toml:"batch_size"` // Порог СЗn (п.4.1 ТЗ): количество записей для пакетной обработки в list
	Timeout     time.Duration `toml:"timeout"`    // Таймер ТПn (п.4.2 ТЗ): время ожидания для обработки записей, если порог не достигнут
}

// PostgreSQLConfig содержит параметры подключения к СУБД PostgreSQL (раздел PostgreSQL по п.1.11 ТЗ).
type PostgreSQLConfig struct {
	Host     string `toml:"host"`     // IP или хост подключения к целевой СУБД
	Port     int    `toml:"port"`     // Порт подключения (обычно 5432)
	User     string `toml:"user"`     // Имя пользователя СУБД
	Password string `toml:"password"` // Пароль
	Database string `toml:"database"` // Имя целевой базы данных ("Venera" по п.3 ТЗ)
	SSLMode  string `toml:"ssl_mode"` // Режим SSL (disable/require)
}

// SystemConfig содержит параметры системных ограничений для защиты от переполнения (п.17 ТЗ).
type SystemConfig struct {
	DiskWarningThreshold  float64 `toml:"disk_warning_threshold"`  // Процент (N), при котором выдавать предупреждение (п.17.1 ТЗ)
	DiskCriticalThreshold float64 `toml:"disk_critical_threshold"` // Процент (M), при котором останавливать процессы (п.17.2 ТЗ)
	RamCriticalThreshold  float64 `toml:"ram_critical_threshold"`  // Процент свободной RAM (L) для принудительной остановки (п.17.2 ТЗ)
}

