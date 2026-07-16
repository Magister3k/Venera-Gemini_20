package models

// ProcessSourceType определяет тип источника данных
type ProcessSourceType string

const (
	SourceNetwork ProcessSourceType = "network" // Сетевая карта (непрерывный поток)
	SourceFolder  ProcessSourceType = "folder"  // Папка с файлами pcap
	SourceFile    ProcessSourceType = "file"    // Отдельный pcap-файл
)

// ProcessStatus определяет текущее состояние процесса.
type ProcessStatus string

const (
	StatusStopped ProcessStatus = "stopped" // Процесс остановлен
	StatusRunning ProcessStatus = "running" // Процесс запущен и собирает данные (ПВn)
	StatusError   ProcessStatus = "error"   // Процесс остановлен из-за ошибки
)

// ProcessConfig описывает параметры отдельного процесса сбора данных.
// Эти данные хранятся в файле processes.toml с группировкой по ID процесса (п.1.9 ТЗ).
type ProcessConfig struct {
	ID                 string            `toml:"id"`                           // Уникальный идентификатор процесса
	Type               ProcessSourceType `toml:"type"`                         // Тип источника (network, folder, file)
	Name               string            `toml:"name"`                         // Пользовательское название источника (source)
	IP                 string            `toml:"ip,omitempty"`                 // Для SourceNetwork: IP-адрес сетевой карты
	UDPPort            int               `toml:"udp_port,omitempty"`           // Для SourceNetwork: прослушиваемый UDP-порт
	FolderPath         string            `toml:"folder_path,omitempty"`        // Для SourceFolder: путь к папке с файлами
	ScanSubfolders     bool              `toml:"scan_subfolders,omitempty"`    // Для SourceFolder: режим сканирования подпапок (п.13 ТЗ)
	MonitorNewFiles    bool              `toml:"monitor_new_files,omitempty"`  // Для SourceFolder: режим мониторинга новых файлов (п.13 ТЗ)
	FilePath           string            `toml:"file_path,omitempty"`          // Для SourceFile: абсолютный путь к отдельному файлу
	Status             ProcessStatus     `toml:"-"`                            // Текущий статус (не сохраняется в toml)
}

// ProcessesFile структура для хранения процессов в корневом TOML файле (processes.toml).
type ProcessesFile struct {
	Processes map[string]ProcessConfig `toml:"processes"`
}

// DataEntry представляет одну разобранную запись (ключ-значение-время), готовую для вставки в СУБД.
// Соответствует структуре базы данных "Venera" СУБД PostgreSQL (п.3 ТЗ).
type DataEntry struct {
	Source    string // Название источника (source)
	Key       string // Ключ (key)
	Value     string // Значение (value)
	Timestamp int64  // Unix timestamp (мс) (используется для date_first и date_last через UPSERT)
}

