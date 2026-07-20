package models

// ProcessSourceType определяет тип источника данных
type ProcessSourceType string
const (
	SourceNetwork ProcessSourceType = "network" // Сетевая карта (непрерывный поток)
	SourceFolder  ProcessSourceType = "folder"  // Папка с файлами pcap
	SourceFile    ProcessSourceType = "file"    // Отдельный файл pcap
)

// ProcessStatus определяет текущее состояние процесса.
type ProcessStatus string
const (
	StatusStopped ProcessStatus = "stopped" // Процесс остановлен
	StatusRunning ProcessStatus = "running" // Процесс запущен и собирает данные
	StatusError   ProcessStatus = "error"   // Процесс остановлен из-за ошибки
)

// ProcessConfig описывает параметры отдельного процесса сбора данных.
// Эти данные хранятся в файле processes.toml с группировкой по ID процесса.
type ProcessConfig struct {
	ID              string            `toml:"id"`                          // Уникальный идентификатор процесса
	Type            ProcessSourceType `toml:"type"`                        // Тип источника данных
	Name            string            `toml:"name"`                        // Пользовательское название источника
	IP              string            `toml:"ip,omitempty"`                // Для SourceNetwork: IP-адрес сетевой карты
	UDPPort         int               `toml:"udp_port,omitempty"`          // Для SourceNetwork: прослушиваемый UDP-порт
	FolderPath      string            `toml:"folder_path,omitempty"`       // Для SourceFolder: путь к папке с файлами
	ScanSubfolders  bool              `toml:"scan_subfolders,omitempty"`   // Для SourceFolder: режим сканирования подпапок
	MonitorNewFiles bool              `toml:"monitor_new_files,omitempty"` // Для SourceFolder: режим мониторинга новых файлов
	FilePath        string            `toml:"file_path,omitempty"`         // Для SourceFile: абсолютный путь к отдельному файлу
	Status          ProcessStatus     `toml:"-"`                           // Текущий статус (не сохраняется в processes.toml)
}

// ProcessesFile структура для хранения процессов в файле формата TOML (processes.toml).
type ProcessesFile struct {
	Processes map[string]ProcessConfig `toml:"processes"`
}

// DataEntry представляет одну разобранную запись (ключ-значение-время), готовую для вставки в итоговую базу.
type DataEntry struct {
	Source    string // Название источника
	Key       string // Ключ
	Value     string // Значение
	Timestamp int64  // Unix timestamp (мс) (используется для date_first и date_last через UPSERT)
}
