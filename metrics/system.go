package metrics

import (
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

// SystemStats хранит общую статистику
type SystemStats struct {
	FreeRAMBytes   uint64
	FreeRAMPercent float64
	DbDiskFree     uint64
	DbDiskPercent  float64
}

// GetSystemStats собирает статистику (п.6.1)
func GetSystemStats(dbPath string) (*SystemStats, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}

	stats := &SystemStats{
		FreeRAMBytes:   v.Available,
		FreeRAMPercent: 100.0 - v.UsedPercent,
	}

	if dbPath != "" {
		usage, err := disk.Usage(dbPath)
		if err == nil {
			stats.DbDiskFree = usage.Free
			stats.DbDiskPercent = 100.0 - usage.UsedPercent
		}
	}

	return stats, nil
}

// ProtectSystem проверяет пороги и вызывает действия (п.17)
func ProtectSystem(stats *SystemStats, warningThresh, criticalDiskThresh, criticalRamThresh float64) (bool, string) {
	if stats.DbDiskPercent < criticalDiskThresh {
		return true, "Критически мало места на диске!"
	}
	if stats.FreeRAMPercent < criticalRamThresh {
		return true, "Критически мало свободной оперативной памяти!"
	}
	if stats.DbDiskPercent < warningThresh {
		// Просто предупреждение (нужно отправить в UI/SysTray)
	}
	return false, ""
}
