package models

// SystemMetrics описывает общую системную статистику.
type SystemMetrics struct {
	FreeRAMBytes      uint64  `json:"free_ram_bytes"`
	FreeRAMPercent    float64 `json:"free_ram_percent"`
	DbDiskFreeBytes   uint64  `json:"db_disk_free_bytes"`
	DbDiskFreePercent float64 `json:"db_disk_free_percent"`
	DragonflyDBSize   uint64  `json:"dragonfly_db_size"` // Размер кэширующей СУБД
	PostgreSQLSize    uint64  `json:"postgresql_size"`   // Размер базы в СУБД PostgreSQL
}

// ProcessMetrics описывает статистику по каждому рабочему процессу.
type ProcessMetrics struct {
	ProcessID       string  `json:"process_id"`
	SourceName      string  `json:"source_name"`
	InputSpeedBps   float64 `json:"input_speed_bps"`   // Скорость входного потока в Tshark (байт/с)
	RamConsumption  uint64  `json:"ram_consumption"`   // Потребление RAM процессом (байты)
	CpuLoadPercent  float64 `json:"cpu_load_percent"`  // Загрузка CPU процессом (%)
	TotalPairs      int64   `json:"total_pairs"`       // Общее количество пар ключ-значение (из входного json от TShark)
	FilteredPairs   int64   `json:"filtered_pairs"`    // Количество отобранных пар ключ-значение (помещенных в list кэширующей СУБД)
	UniquePairsSent int64   `json:"unique_pairs_sent"` // Количество уникальных пар ключ-значение (добавленных в итоговую базу)
}

// StatsPayload используется для отправки метрик через WebSocket в UI.
type StatsPayload struct {
	System    SystemMetrics             `json:"system"`
	Processes map[string]ProcessMetrics `json:"processes"`
}
