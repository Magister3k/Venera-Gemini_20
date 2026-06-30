package models

// ProcessSourceType определяет тип источника данных
type ProcessSourceType string

const (
	SourceNetwork ProcessSourceType = "network"
	SourceFolder  ProcessSourceType = "folder"
	SourceFile    ProcessSourceType = "file"
)

// ProcessConfig описывает параметры отдельного процесса сбора данных.
// Эти данные хранятся в processes.toml.
type ProcessConfig struct {
	ID                 string            `toml:"id"`
	Type               ProcessSourceType `toml:"type"`
	Name               string            `toml:"name"`
	IP                 string            `toml:"ip,omitempty"`           // Для SourceNetwork
	UDPPort            int               `toml:"udp_port,omitempty"`     // Для SourceNetwork
	FolderPath         string            `toml:"folder_path,omitempty"`  // Для SourceFolder
	ScanSubfolders     bool              `toml:"scan_subfolders,omitempty"`// Для SourceFolder
	MonitorNewFiles    bool              `toml:"monitor_new_files,omitempty"`// Для SourceFolder
	FilePath           string            `toml:"file_path,omitempty"`    // Для SourceFile
	Status             string            `toml:"-"`                      // Текущий статус (не сохраняется в toml) "stopped", "running", "error"
}

// ProcessesFile структура для хранения мапы процессов
type ProcessesFile struct {
	Processes map[string]ProcessConfig `toml:"processes"`
}

// DataEntry представляет одну разобранную запись (ключ-значение-время)
type DataEntry struct {
	Source    string // ID источника
	Key       string
	Value     string
	Timestamp int64 // Unix timestamp (мс)
}
