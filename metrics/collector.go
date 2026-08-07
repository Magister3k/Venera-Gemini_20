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
		Sys:    models.SysMetrics{},
		Procs: GetProcsMetrics(),
	}

	// Сбор памяти
	freeRAMBytes, freeRAMPercent, err := GetSysRAM()
	if err == nil {
		payload.Sys.FreeRAMBytes = freeRAMBytes
		payload.Sys.FreeRAMPercent = freeRAMPercent
	}

	// Сбор диска (берем диск, где PostgreSQL)
	// В реальной системе нужно извлечь букву диска из конфига.
	// Здесь берем "C:\" как резерв.
	dbPath := "C:\\"
	freeDiskBytes, freeDiskPercent, err := GetDiskSpace(dbPath)
	if err == nil {
		payload.Sys.DbDiskFreeBytes = freeDiskBytes
		payload.Sys.DbDiskFreePercent = freeDiskPercent
	}

	// Сбор размеров БД (кэширующая СУБД и итоговая БД)
	cacheDbSize, pgSize, _ := GetDbSizes()
	payload.Sys.CacheDbSize = cacheDbSize
	payload.Sys.PgDbSize = pgDbSize

	return payload
}

// GetSysStatsCompat - адаптер для поддержки вызова ProtectSystem
func GetSysStatsCompat() *SystemStats {
	freeRAMBytes, freeRAMPercent, _ := GetSysRAM()

	dbPath := "C:\\"
	_, freeDiskPercent, _ := GetDiskSpace(dbPath)

	return &SysStats{
		FreeRAMBytes:   freeRAMBytes,
		FreeRAMPercent: freeRAMPercent,
		DbDiskPercent:  freeDiskPercent,
	}
}

// SysStats структура для совместимости с ProtectSystem
type SysStats struct {
	FreeRAMBytes   uint64
	FreeRAMPercent float64
	DbDiskPercent  float64
}

// ProtectSys проверяет пороги и возвращает флаг необходимости остановки процессов.
func ProtectSys() (bool, string) {
	stats := GetSysStatsCompat()

	cfg := config.GlobalCfg.System

	if stats.DbDiskPercent < cfg.DiskCriticalThreshold {
		return true, "Критически мало свободного места на диске! Остановка всех процессов."
	}
	if stats.FreeRAMPercent < cfg.RamCriticalThreshold {
		return true, "Критически мало свободной оперативной памяти! Остановка всех процессов."
	}

	return false, ""
}

// CheckDiskWarn проеряет порог и возвращает флаг необходимости предупредить
// пользователя о подходе объема свободного места на диске к критической отметке.
func CheckDiskWarn() bool {
	stats := GetSysStatsCompat()
	cfg := config.GlobalCfg.System

	// Если мы уже в критической зоне, возвращаем false, так как сработает ProtectSystem
	if stats.DbDiskPercent < cfg.DiskCriticalThreshold {
		return false
	}

	return stats.DbDiskPercent < cfg.DiskWarningThreshold
}
