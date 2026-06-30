package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"venera/models"
)

var (
	GlobalConfig *models.Config
	ConfigPath   = "config.toml"
)

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() *models.Config {
	return &models.Config{
		Generic: models.GenericConfig{
			Mode:             "tray",
			AutoStart:        false,
			MaxProcesses:     20,
			WebServerPort:    8080,
			LogRotationDays:  7,
		},
		Paths: models.PathsConfig{
			PodmanExe:   "podman", // или полный путь C:\\Program Files\\...
			TsharkExe:   "tshark", // или полный путь
			FilterList:  "settings/generic.flt",
			ControlList: "settings/generic.ctr",
			AlertsList:  "settings/generic.alr",
			DbImage:     "docker.io/dragonflydb/dragonfly",
			DbBackupDir: "backups",
		},
		DragonflyDB: models.DragonflyDBConfig{
			Host:      "127.0.0.1",
			Port:      6379,
			Password:  "",
			BatchSize: 1000,
			Timeout:   5 * time.Second, // Например, 5 секунд
		},
		PostgreSQL: models.PostgreSQLConfig{
			Host:     "127.0.0.1",
			Port:     5432,
			User:     "postgres",
			Password: "password",
			Database: "Venera",
			SSLMode:  "disable",
		},
		System: models.SystemConfig{
			DiskWarningThreshold:  15.0, // 15% свободно
			DiskCriticalThreshold: 5.0,  // 5% свободно
			RamCriticalThreshold:  5.0,  // 5% свободно
		},
	}
}

// LoadConfig загружает конфигурацию из файла config.toml
func LoadConfig() error {
	GlobalConfig = DefaultConfig()

	if _, err := os.Stat(ConfigPath); os.IsNotExist(err) {
		// Если файла нет, создаем его с дефолтными настройками
		return SaveConfig(GlobalConfig)
	}

	_, err := toml.DecodeFile(ConfigPath, GlobalConfig)
	if err != nil {
		return fmt.Errorf("ошибка парсинга config.toml: %v", err)
	}

	// Создаем папку для бэкапов, если не существует
	if GlobalConfig.Paths.DbBackupDir != "" {
		_ = os.MkdirAll(GlobalConfig.Paths.DbBackupDir, 0755)
	}

	return nil
}

// SaveConfig сохраняет текущую конфигурацию в файл
func SaveConfig(cfg *models.Config) error {
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

// EnsureDir проверяет существование папки для файла и создает её
func EnsureDir(fileName string) {
	dir := filepath.Dir(fileName)
	if dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0755)
	}
}
