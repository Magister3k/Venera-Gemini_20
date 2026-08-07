package metrics

import (
	"sync"

	"venera/models"
)

var (
	// procStats хранит статистику по процессам в реальном времени.
	// Ключ - ID процесса.
	procStats = make(map[string]*models.ProcMetrics)
	procStatsMu  sync.RWMutex
)

// UpdProcSpeed обновляет скорость входного потока (байт/с) для процесса
func UpdProcSpeed(id string, speed float64) {
	procStatsMu.Lock()
	defer procStatsMu.Unlock()
	if _, ok := procStats[id]; !ok {
		procStats[id] = &models.ProcMetrics{ProcID: id}
	}
	procStats[id].InputSpeedBps = speed
}

// UpdProcResource обновляет данные RAM и CPU для процесса
func UpdProcResource(id string, ram uint64, cpu float64) {
	procStatsMu.Lock()
	defer procStatsMu.Unlock()
	if _, ok := procStats[id]; !ok {
		procStats[id] = &models.ProcMetrics{ProcID: id}
	}
	procStats[id].RamConsumption = ram
	procStats[id].CpuLoadPercent = cpu
}

// IncProcCounts обновляет счетчики обработанных сообщений
func IncProcCounts(id string, total, filtered, unique int64) {
	procStatsMu.Lock()
	defer procStatsMu.Unlock()
	if _, ok := procStats[id]; !ok {
		procStats[id] = &models.ProcMetrics{ProcID: id}
	}
	procStats[id].TotalPairs += total
	procStats[id].FilteredPairs += filtered
	procStats[id].UniquePairs += unique
}

// GetProcMetrics возвращает копию метрик процесса
func GetProcMetrics() map[string]models.ProcMetrics {
	procStatsMu.RLock()
	defer procStatsMu.RUnlock()

	copyMap := make(map[string]models.ProcMetrics)
	for k, v := range procStats {
		copyMap[k] = *v
	}
	return copyMap
}

// ClearProcMetrics удаляет метрики процесса (при его удалении)
func ClearProcMetrics(id string) {
	procStatsMu.Lock()
	defer procStatsMu.Unlock()
	delete(procStats, id)
}
