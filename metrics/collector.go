package metrics

import (
	"venera/config"
	"venera/models"
)

// CollectAllStats собирает всю систему метрик
// для передачи на фронтенд (StatsPayload)
// или для вывода в Zabbix.
func CollectAllStats() models.StatsPayload {
	payload := models.StatsPayload{
		System:    models.SystemMetrics{},
		Processes: GetProcessMetrics(),
	}

	// Сбор памяти
	freeRAMBytes, freeRAMPercent, err := GetSystemRAM()
	if err == nil {
		payload.System.FreeRAMBytes = freeRAMBytes
		payload.System.FreeRAMPercent = freeRAMPercent
	}

	// Сбор диска (берем диск, где PostgreSQL)
	// В реальной системе нужно извлечь букву диска из конфига.
	// Здесь берем "C:\" как резерв.
	dbPath := "C:\\"
	freeDiskBytes, freeDiskPercent, err := GetDiskSpace(dbPath)
	if err == nil {
		payload.System.DbDiskFreeBytes = freeDiskBytes
		payload.System.DbDiskFreePercent = freeDiskPercent
	}

	// Сбор размеров БД (DragonflyDB и PostgreSQL)
	dragonflySize, postgresSize, _ := GetDatabaseSizes()
	payload.System.DragonflyDBSize = dragonflySize
	payload.System.PostgreSQLSize = postgresSize

	return payload
}

// GetSystemStatsCompat - адаптер для поддержки вызова ProtectSystem
func GetSystemStatsCompat() *SystemStats {
	freeRAMBytes, freeRAMPercent, _ := GetSystemRAM()

	dbPath := "C:\\"
	_, freeDiskPercent, _ := GetDiskSpace(dbPath)

	return &SystemStats{
		FreeRAMBytes:   freeRAMBytes,
		FreeRAMPercent: freeRAMPercent,
		DbDiskPercent:  freeDiskPercent,
	}
}

// SystemStats структура для совместимости с ProtectSystem
type SystemStats struct {
	FreeRAMBytes   uint64
	FreeRAMPercent float64
	DbDiskPercent  float64
}

// ProtectSystem проверяет пороги и возвращает флаг необходимости остановки процессов.
func ProtectSystem() (bool, string) {
	stats := GetSystemStatsCompat()

	cfg := config.GlobalConfig.System

	if stats.DbDiskPercent < cfg.DiskCriticalThreshold {
		return true, "Критически мало свободного места на диске! Остановка всех процессов."
	}
	if stats.FreeRAMPercent < cfg.RamCriticalThreshold {
		return true, "Критически мало свободной оперативной памяти! Остановка всех процессов."
	}

	return false, ""
}

// CheckDiskWarning проеряет порог и возвращает флаг необходимости предупредить
// пользователя о подходе объема свободного места на диске к критической отметке.
func CheckDiskWarning() bool {
	stats := GetSystemStatsCompat()
	cfg := config.GlobalConfig.System

	// Если мы уже в критической зоне, возвращаем false, так как сработает ProtectSystem
	if stats.DbDiskPercent < cfg.DiskCriticalThreshold {
		return false
	}

	return stats.DbDiskPercent < cfg.DiskWarningThreshold
}
