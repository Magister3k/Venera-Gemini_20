package metrics

import (
	"venera/config"
	"venera/models"
)

// CollectAllStats собирает всю систему метрик
// для передачи в веб-интерфейс или вывода в Zabbix.
func CollectAllStats() models.StatsPayload {
	payload := models.StatsPayload{
		Sys:   models.SysMetrics{},
		Procs: GetProcsMetrics(),
	}

	// Сбор информации о памяти
	freeRAMBytes, freeRAMPerc, err := GetSysRAM()
	if err == nil {
		payload.Sys.FreeRAMBytes = freeRAMBytes
		payload.Sys.FreeRAMPerc = freeRAMPerc
	}

	// Сбор информации о диске с итоговой БД
	// В реальной системе нужно извлечь букву диска из конфига.
	// Здесь берем "C:\" как резерв.
	dbPath := "C:\\"
	freeDiskBytes, freeDiskPerc, err := GetDiskSpace(dbPath)
	if err == nil {
		payload.Sys.DbDiskFreeBytes = freeDiskBytes
		payload.Sys.DbDiskFreePerc = freeDiskPerc
	}

	// Сбор размеров кэширующей СУБД и итоговой БД)
	cacheDbSize, pgDbSize, _ := GetDbSizes()
	payload.Sys.CacheDbSize = cacheDbSize
	payload.Sys.PgDbSize = pgDbSize

	return payload
}

// GetSysStatsCompat - адаптер для поддержки вызова ProtectSystem
func GetSysStatsCompat() *SysStats {
	freeRAMBytes, freeRAMPerc, _ := GetSysRAM()

	dbPath := "C:\\"
	_, freeDiskPerc, _ := GetDiskSpace(dbPath)

	return &SysStats{
		FreeRAMBytes:   freeRAMBytes,
		FreeRAMPerc: freeRAMPerc,
		DbDiskPerc:  freeDiskPerc,
	}
}

// SysStats структура для совместимости с ProtectSystem
type SysStats struct {
	FreeRAMBytes   uint64
	FreeRAMPerc float64
	DbDiskPerc  float64
}

// ProtectSystem проверяет пороги и возвращает флаг необходимости остановки процессов.
func ProtectSystem() (bool, string) {
	stats := GetSysStatsCompat()

	cfg := config.GlobalCfg.System

	if stats.DbDiskPerc < cfg.DiskCriticalThreshold {
		return true, "Критически мало свободного места на диске! Остановка всех процессов."
	}
	if stats.FreeRAMPerc < cfg.RamCriticalThreshold {
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
	if stats.DbDiskPerc < cfg.DiskCriticalThreshold {
		return false
	}

	return stats.DbDiskPerc < cfg.DiskWarningThreshold
}
