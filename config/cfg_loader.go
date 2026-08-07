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
	// mu защищает доступ к GlobalCfg для предотвращения Race Conditions
	mu           sync.RWMutex
	GlobalCfg *models.Config
	CfgPath   string
)

func init() {
	// Определение пути к config.toml относительно исполняемого файла,
	// что критически важно при запуске в качестве службы Windows,
	// так как рабочий каталог службы может отличаться (например, C:\Windows\System32).
	exePath, err := os.Executable()
	if err == nil {
		CfgPath = filepath.Join(filepath.Dir(exePath), "config.toml")
	} else {
		// Fallback
		CfgPath = "config.toml"
	}
}

// DefaultCfg возвращает конфигурацию по умолчанию
func DefaultCfg() *models.Config {
	return &models.Config{
		Generic: models.GenericCfg{
			Mode:                 "tray",
			AutoStart:            false,
			MaxProcs:             20,
			WebSrvPort:           8080,
			LogRotDays:           7,
			LocalUIType:          "react",
			ShowConsoleOnStartup: true,
		},
		Paths: models.PathsCfg{
			Podman:    "progs/podman/podman.exe",
			Tshark:    "progs/wireshark/tshark.exe",
			Filter:   "settings/generic.flt",
			Control:  "settings/generic.ctr",
			Alerts:   "settings/generic.alr",
			CacheDbImage: "progs/dragonflydb/dragonflydb.tar.gz",
			CacheDir:     "cache",
		},
		CacheDb: models.CacheDbCfg{
			Host:      "127.0.0.1",
			Port:      6379,
			Pass:      "",
			BatchSize: 1000,
			Timeout:   5 * time.Second,
		},
		PgDb: models.PgDbCfg{
			Host:     "127.0.0.1",
			Port:     5432,
			User:     "postgres",
			Pass:     "postgres",
			Name:     "Venera",
			SSLMode:  "disable",
		},
		System: models.SysCfg{
			DiskWarningThreshold:  15.0, // Объем свободного места для предупреждения
			DiskCriticalThreshold: 5.0,  // Объем свободного места для остановки процессов
			RamCriticalThreshold:  5.0,  // Объем свободной RAM для остановки процессов
		},
	}
}

// GetCfg возвращает потокобезопасную глубокую копию или указатель на текущую конфигурацию.
// Используется для предотвращения race condition при чтении из разных горутин.
func GetCfg() models.Config {
	mu.RLock()
	defer mu.RUnlock()
	if GlobalCfg == nil {
		return *DefaultCfg()
	}
	return *GlobalCfg
}

// LoadCfg загружает конфигурацию из файла config.toml с валидацией параметров
func LoadCfg() error {
	mu.Lock()
	defer mu.Unlock()

	tempCfg := DefaultCfg()

	if _, err := os.Stat(CfgPath); os.IsNotExist(err) {
		// Если файл не существует, создаем директорию и записываем дефолтную конфигурацию
		EnsureDir(CfgPath)
		GlobalCfg = tempCfg
		mu.Unlock() // временно отпускаем мьютекс для SaveCfg, так как он сам захватит его
		errSave := SaveCfg(tempCfg)
		mu.Lock() // возвращаем блокировку
		if errSave != nil {
			return fmt.Errorf("не удалось создать конфигурацию по умолчанию: %v", errSave)
		}
		return nil
	}

	// Декодируем TOML во временную структуру, чтобы не повредить GlobalCfg при ошибках парсинга
	_, err := toml.DecodeFile(CfgPath, tempCfg)
	if err != nil {
		return fmt.Errorf("ошибка парсинга config.toml: %v", err)
	}

	// Валидация параметров
	if err := validateCfg(tempCfg); err != nil {
		return fmt.Errorf("валидация конфигурации не пройдена: %v", err)
	}

	GlobalCfg = tempCfg

	// Применяем зеркалирование (разрешение) путей относительно папки с исполняемым файлом.
	// Это гарантирует, что пути к настройкам, бэкапам и логам не зависят от рабочей директории службы.
	exeDir := filepath.Dir(CfgPath)
	
	resolvePath := func(p *string) {
		if *p != "" && !filepath.IsAbs(*p) {
			*p = filepath.Join(exeDir, *p)
		}
	}

	resolvePath(&GlobalCfg.Paths.Filter)
	resolvePath(&GlobalCfg.Paths.Control)
	resolvePath(&GlobalCfg.Paths.Alerts)
	resolvePath(&GlobalCfg.Paths.CacheDir)

	// Создаем папку для резервирования кэширубщей СУБД, если указана
	if GlobalCfg.Paths.CacheDir != "" {
		_ = os.MkdirAll(GlobalCfg.Paths.CacheDir, 0755)
	}

	return nil
}

// UpdateCfg обновляет текущую конфигурацию в памяти и сохраняет ее на диск атомарно
func UpdateCfg(cfg *models.Config) error {
	if err := validateCfg(cfg); err != nil {
		return fmt.Errorf("валидация обновленной конфигурации не пройдена: %v", err)
	}

	mu.Lock()
	GlobalCfg = cfg
	mu.Unlock()

	return SaveCfg(cfg)
}

// SaveCfg сохраняет конфигурацию в файл config.toml с созданием резервной копии (.bak) для надежности
func SaveCfg(cfg *models.Config) error {
	mu.Lock()
	defer mu.Unlock()

	// Если файл уже существует, создаем его резервную копию перед перезаписью
	if _, err := os.Stat(CfgPath); err == nil {
		bakPath := CfgPath + ".bak"
		_ = os.Remove(bakPath) // Удаляем старый бэкап, если он был
		_ = os.Rename(CfgPath, bakPath)
	}

	EnsureDir(CfgPath)
	file, err := os.Create(CfgPath)
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

// validateCfg проверяет корректность параметров конфигурации
func validateCfg(cfg *models.Config) error {
	// Проверка режима работы
	if cfg.Generic.Mode != "tray" && cfg.Generic.Mode != "service" {
		return fmt.Errorf("недопустимый режим работы '%s'; разрешены только 'tray' или 'service'", cfg.Generic.Mode)
	}

	// Проверка портов веб-сервера
	if cfg.Generic.WebSrvPort <= 0 || cfg.Generic.WebSrvPort > 65535 {
		return fmt.Errorf("недопустимый порт веб-сервера: %d", cfg.Generic.WebSrvPort)
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

	// Проверка лимитов кэширующей СУБД
	if cfg.CacheDb.BatchSize <= 0 {
		return fmt.Errorf("размер пакета обработки кэшируюшей СУБД должен быть больше 0")
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
