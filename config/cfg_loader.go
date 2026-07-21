package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"venera/models"
)

var (
	// mu защищает доступ к GlobalConfig для предотвращения Race Conditions
	mu           sync.RWMutex
	GlobalConfig *models.Config
	ConfigPath   string
)

func init() {
	// Определение пути к config.toml относительно исполняемого файла,
	// что критически важно при запуске в качестве службы Windows,
	// так как рабочий каталог службы может отличаться (например, C:\Windows\System32).
	exePath, err := os.Executable()
	if err == nil {
		ConfigPath = filepath.Join(filepath.Dir(exePath), "config.toml")
	} else {
		// Fallback
		ConfigPath = "config.toml"
	}
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() *models.Config {
	return &models.Config{
		Generic: models.GenericConfig{
			Mode:                 "tray", // Режим работы: tray (системный трей) или service (служба)
			AutoStart:            false,  // Автоматический старт всех процессов при запуске
			MaxProcesses:         20,     // Ограничение количества процессов
			WebServerPort:        8080,   // Порт веб-сервера по умолчанию
			LogRotationDays:      7,      // Срок хранения логов в днях
			LocalUIType:          "react",
			ShowConsoleOnStartup: true,   // По умолчанию показываем консоль, потом скрываем
		},
		Paths: models.PathsConfig{
			PodmanExe:   "progs/podman/podman.exe",
			TsharkExe:   "progs/wireshark/tshark.exe",
			FilterList:  "settings/generic.flt",
			ControlList: "settings/generic.ctr",
			AlertsList:  "settings/generic.alr",
			DbImage:     "progs/dragonflydb/dragonflydb.tar.gz",
			DbBackupDir: "cache",
		},
		DragonflyDB: models.DragonflyDBConfig{
			Host:      "127.0.0.1",
			Port:      6379,
			Password:  "",
			BatchSize: 1000,
			Timeout:   5 * time.Second,
		},
		PostgreSQL: models.PostgreSQLConfig{
			Host:     "127.0.0.1",
			Port:     5432,
			User:     "postgres",
			Password: "postgres",
			Database: "Venera",
			SSLMode:  "disable",
		},
		System: models.SystemConfig{
			DiskWarningThreshold:  15.0, // Объем свободного места для предупреждения
			DiskCriticalThreshold: 5.0,  // Объем свободного места для остановки процессов
			RamCriticalThreshold:  5.0,  // Объем свободной RAM для остановки процессов
		},
	}
}

// GetConfig возвращает потокобезопасную глубокую копию или указатель на текущую конфигурацию.
// Используется для предотвращения race condition при чтении из разных горутин.
func GetConfig() models.Config {
	mu.RLock()
	defer mu.RUnlock()
	if GlobalConfig == nil {
		return *DefaultConfig()
	}
	return *GlobalConfig
}

// LoadConfig загружает конфигурацию из файла config.toml с валидацией параметров
func LoadConfig() error {
	mu.Lock()
	defer mu.Unlock()

	tempConfig := DefaultConfig()

	if _, err := os.Stat(ConfigPath); os.IsNotExist(err) {
		// Если файл не существует, создаем директорию и записываем дефолтную конфигурацию
		EnsureDir(ConfigPath)
		GlobalConfig = tempConfig
		mu.Unlock() // временно отпускаем мьютекс для SaveConfig, так как он сам захватит его
		errSave := SaveConfig(tempConfig)
		mu.Lock() // возвращаем блокировку
		if errSave != nil {
			return fmt.Errorf("не удалось создать конфигурацию по умолчанию: %v", errSave)
		}
		return nil
	}

	// Декодируем TOML во временную структуру, чтобы не повредить GlobalConfig при ошибках парсинга
	_, err := toml.DecodeFile(ConfigPath, tempConfig)
	if err != nil {
		return fmt.Errorf("ошибка парсинга config.toml: %v", err)
	}

	// Валидация параметров согласно жестким требованиям ТЗ
	if err := validateConfig(tempConfig); err != nil {
		return fmt.Errorf("валидация конфигурации не пройдена: %v", err)
	}

	GlobalConfig = tempConfig

	// Применяем зеркалирование (разрешение) путей относительно папки с исполняемым файлом.
	// Это гарантирует, что пути к настройкам, бэкапам и логам не зависят от рабочей директории службы.
	exeDir := filepath.Dir(ConfigPath)
	
	resolvePath := func(p *string) {
		if *p != "" && !filepath.IsAbs(*p) {
			*p = filepath.Join(exeDir, *p)
		}
	}

	resolvePath(&GlobalConfig.Paths.FilterList)
	resolvePath(&GlobalConfig.Paths.ControlList)
	resolvePath(&GlobalConfig.Paths.AlertsList)
	resolvePath(&GlobalConfig.Paths.DbBackupDir)

	// Создаем папку для бэкапов DragonflyDB, если указана
	if GlobalConfig.Paths.DbBackupDir != "" {
		_ = os.MkdirAll(GlobalConfig.Paths.DbBackupDir, 0755)
	}

	return nil
}

// UpdateConfig обновляет текущую конфигурацию в памяти и сохраняет ее на диск атомарно
func UpdateConfig(cfg *models.Config) error {
	if err := validateConfig(cfg); err != nil {
		return fmt.Errorf("валидация обновленной конфигурации не пройдена: %v", err)
	}

	mu.Lock()
	GlobalConfig = cfg
	mu.Unlock()

	return SaveConfig(cfg)
}

// SaveConfig сохраняет конфигурацию в файл config.toml с созданием резервной копии (.bak) для надежности
func SaveConfig(cfg *models.Config) error {
	mu.Lock()
	defer mu.Unlock()

	// Если файл уже существует, создаем его резервную копию перед перезаписью
	if _, err := os.Stat(ConfigPath); err == nil {
		bakPath := ConfigPath + ".bak"
		_ = os.Remove(bakPath) // Удаляем старый бэкап, если он был
		_ = os.Rename(ConfigPath, bakPath)
	}

	EnsureDir(ConfigPath)
	file, err := os.Create(ConfigPath)
	if err != nil {
		return fmt.Errorf("ошибка создания config.toml: %v", err)
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(cfg); err != nil {
		return fmt.Errorf("ошибка кодирования config.toml: %v", err)
	}

	return nil
}

// validateConfig проверяет корректность параметров конфигурации на соответствие ТЗ
func validateConfig(cfg *models.Config) error {
	// Одновременно может быть запущено до 20 рабочих процессов
	if cfg.Generic.MaxProcesses <= 0 || cfg.Generic.MaxProcesses > 20 {
		return fmt.Errorf("максимальное количество процессов должно быть в диапазоне от 1 до 20 (настроено: %d)", cfg.Generic.MaxProcesses)
	}

	// Проверка режима работы
	if cfg.Generic.Mode != "tray" && cfg.Generic.Mode != "service" {
		return fmt.Errorf("недопустимый режим работы '%s'; разрешены только 'tray' или 'service'", cfg.Generic.Mode)
	}

	// Проверка портов веб-сервера
	if cfg.Generic.WebServerPort <= 0 || cfg.Generic.WebServerPort > 65535 {
		return fmt.Errorf("недопустимый порт веб-сервера: %d", cfg.Generic.WebServerPort)
	}

	// Проверка порогов диска и памяти
	if cfg.System.DiskWarningThreshold < 0 || cfg.System.DiskWarningThreshold > 100 {
		return fmt.Errorf("порог предупреждения диска должен быть от 0 до 100%%")
	}
	if cfg.System.DiskCriticalThreshold < 0 || cfg.System.DiskCriticalThreshold > 100 {
		return fmt.Errorf("критический порог диска должен быть от 0 до 100%%")
	}
	if cfg.System.RamCriticalThreshold < 0 || cfg.System.RamCriticalThreshold > 100 {
		return fmt.Errorf("критический порог RAM должен быть от 0 до 100%%")
	}

	// Проверка лимитов DragonflyDB
	if cfg.DragonflyDB.BatchSize <= 0 {
		return fmt.Errorf("размер пакета DragonflyDB (batch_size) должен быть больше 0")
	}

	return nil
}

// EnsureDir проверяет существование папки для файла и создает её
func EnsureDir(fileName string) {
	dir := filepath.Dir(fileName)
	if dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0755)
	}
}
