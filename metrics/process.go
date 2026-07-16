package metrics

import (
	"sync"
	"venera/models"
)

var (
	// processStats хранит статистику по процессам в реальном времени.
	// Ключ - ID процесса.
	processStats = make(map[string]*models.ProcessMetrics)
	procStatsMu  sync.RWMutex
)

// UpdateProcessSpeed обновляет скорость входного потока (байт/с) для процесса (п.6.2 ТЗ)
func UpdateProcessSpeed(id string, speed float64) {
	procStatsMu.Lock()
	defer procStatsMu.Unlock()
	if _, ok := processStats[id]; !ok {
		processStats[id] = &models.ProcessMetrics{ProcessID: id}
	}
	processStats[id].InputSpeedBps = speed
}

// UpdateProcessResource обновляет данные RAM и CPU для процесса (п.6.2 ТЗ)
func UpdateProcessResource(id string, ram uint64, cpu float64) {
	procStatsMu.Lock()
	defer procStatsMu.Unlock()
	if _, ok := processStats[id]; !ok {
		processStats[id] = &models.ProcessMetrics{ProcessID: id}
	}
	processStats[id].RamConsumption = ram
	processStats[id].CpuLoadPercent = cpu
}

// IncrementProcessCounts обновляет счетчики обработанных пар JSON (п.6.2 ТЗ)
func IncrementProcessCounts(id string, total, filtered, uniqueSent int64) {
	procStatsMu.Lock()
	defer procStatsMu.Unlock()
	if _, ok := processStats[id]; !ok {
		processStats[id] = &models.ProcessMetrics{ProcessID: id}
	}
	processStats[id].TotalPairs += total
	processStats[id].FilteredPairs += filtered
	processStats[id].UniquePairsSent += uniqueSent
}

// GetProcessMetrics возвращает копию метрик процесса
func GetProcessMetrics() map[string]models.ProcessMetrics {
	procStatsMu.RLock()
	defer procStatsMu.RUnlock()

	copyMap := make(map[string]models.ProcessMetrics)
	for k, v := range processStats {
		copyMap[k] = *v
	}
	return copyMap
}

// ClearProcessMetrics удаляет метрики процесса (при его удалении)
func ClearProcessMetrics(id string) {
	procStatsMu.Lock()
	defer procStatsMu.Unlock()
	delete(processStats, id)
}
