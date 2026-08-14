package models

// SysMetrics описывает общую системную статистику.
type SysMetrics struct {
	FreeRAMBytes      uint64  `json:"free_ram_bytes"`
	FreeRAMPerc    float64 `json:"free_ram_percent"`
	DbDiskFreeBytes   uint64  `json:"db_disk_free_bytes"`
	DbDiskFreePerc float64 `json:"db_disk_free_percent"`
	CacheDbSize       uint64  `json:"cachedb_size"` // Размер кэширующей СУБД
	PgDbSize          uint64  `json:"pg_db_size"`   // Размер итоговой базы в СУБД PostgreSQL
}

// ProcMetrics описывает статистику по каждому рабочему процессу.
type ProcMetrics struct {
	ProcID         string  `json:"process_id"`
	SrcName        string  `json:"source_name"`
	InSpeedBps     float64 `json:"input_speed_bps"`   // Скорость входного потока в Tshark (байт/с)
	RamConsumption uint64  `json:"ram_consumption"`   // Потребление RAM процессом (байты)
	CpuLoadPerc float64 `json:"cpu_load_percent"`  // Загрузка CPU процессом (%)
	TotalPairs     int64   `json:"total_pairs"`       // Общее количество пар ключ-значение (из входного json от TShark)
	FilteredPairs  int64   `json:"filtered_pairs"`    // Количество отобранных пар ключ-значение (помещенных в list кэширующей СУБД)
	UniquePairs    int64   `json:"unique_pairs"`      // Количество уникальных пар ключ-значение (добавленных в итоговую базу)
}

// StatsPayload используется для отправки метрик через WebSocket в UI.
type StatsPayload struct {
	Sys   SysMetrics             `json:"system"`
	Procs map[string]ProcMetrics `json:"processes"`
}
