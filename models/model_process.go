package models

// ProcSrcType определяет тип источника данных
type ProcSrcType string
const (
	SrcNetwork ProcSrcType = "network" // Сетевая карта (непрерывный поток)
	SrcDir     ProcSrcType = "dir"     // Папка с файлами pcap
	SrcFile    ProcSrcType = "file"    // Отдельный файл pcap
)

// ProcStatus определяет текущее состояние процесса.
type ProcStatus string
const (
	StatusStopped ProcStatus = "stopped" // Процесс остановлен
	StatusRunning ProcStatus = "running" // Процесс запущен и собирает данные
	StatusError   ProcStatus = "error"   // Процесс остановлен из-за ошибки
)

// ProcConfig описывает параметры отдельного процесса сбора данных.
// Эти данные хранятся в файле processes.toml с группировкой по ID процесса.
type ProcConfig struct {
	ID              string            `toml:"id"`                          // Уникальный идентификатор процесса
	Type            ProcSrcType       `toml:"type"`                        // Тип источника данных
	Name            string            `toml:"name"`                        // Пользовательское название источника
	IP              string            `toml:"ip,omitempty"`                // Для SrcNetwork: IP-адрес сетевой карты
	UDPPort         int               `toml:"udp_port,omitempty"`          // Для SrcNetwork: прослушиваемый UDP-порт
	DirPath         string            `toml:"dir_path,omitempty"`          // Для SrcDir: путь к папке с файлами
	ScanSubdirs     bool              `toml:"scan_subdirs,omitempty"`      // Для SrcDir: режим сканирования подпапок
	MonitorNewFiles bool              `toml:"monitor_new_files,omitempty"` // Для SrcDir: режим мониторинга новых файлов
	FilePath        string            `toml:"file_path,omitempty"`         // Для SrcFile: абсолютный путь к отдельному файлу
	Status          ProcStatus        `toml:"-"`                           // Текущий статус (не сохраняется в processes.toml)
}

// ProcFile структура для хранения процессов в файле формата TOML (processes.toml).
type ProcFile struct {
	Procs map[string]ProcConfig `toml:"processes"`
}
