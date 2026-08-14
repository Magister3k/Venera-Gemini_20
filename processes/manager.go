package processes

import (
	"context"
	"fmt"
	"sync"
	"time"

	"venera/config"
	"venera/data"
	"venera/logging"
	"venera/models"
)

var (
	Manager = NewProcManager()
)

// RunningProc описывает состояние запущенного процесса
type RunningProc struct {
	Cfg     models.ProcCfg
	Cancel  context.CancelFunc
	Trigger chan struct{} // Канал для сигнализации о достижении порогового значения
	WG      sync.WaitGroup
}

type ProcManager struct {
	mu            sync.Mutex
	activeProcs   map[string]*RunningProc
	activeWorkers int
	workerMu      sync.Mutex

	// srcLocks предотвращает одновременный запуск нескольких воркеров для одного источника
	srcLocks   map[string]bool
	srcLocksMu sync.Mutex
}

func NewProcManager() *ProcManager {
	return &ProcManager{
		activeProcs: make(map[string]*RunningProc),
		srcLocks: make(map[string]bool),
	}
}

// StartProc запускает процесс сбора данных
func (pm *ProcManager) StartProc(p models.ProcCfg) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, exists := pm.activeProcs[p.ID]; exists {
		return fmt.Errorf("процесс %s уже запущен", p.ID)
	}

	ctx, cancel := context.WithCancel(context.Background())

	rp := &RunningProc{
		Cfg:  p,
		Cancel:  cancel,
		Trigger: make(chan struct{}, 1), // Буферизованный канал на 1 сигнал для избежания deadlock
	}

	pm.activeProcs[p.ID] = rp

	UpdProcStatus(p.ID, models.StatusRunning)
	logging.Log.Infof("Запущен процесс: %s (%s)", p.Name, p.ID)

	rp.WG.Add(1)
	go pm.runCollectProc(ctx, rp)

	rp.WG.Add(1)
	go pm.runTaskMgmtProc(ctx, rp)

	return nil
}

// StopProc останавливает процесс
func (pm *ProcManager) StopProc(id string) error {
	pm.mu.Lock()
	rp, exists := pm.activeProcs[id]
	if !exists {
		pm.mu.Unlock()
		return fmt.Errorf("процесс %s не запущен", id)
	}

	rp.Cancel() // Отменяем контекст (завершает Tshark и горутины)
	delete(pm.activeProcs, id)
	pm.mu.Unlock()

	// Ожидаем завершения горутин этого процесса
	rp.WG.Wait()

	UpdProcStatus(id, models.StatusStopped)
	logging.Log.Infof("Остановлен процесс: %s", id)

	// Запускаем финальную обработку остатков в БД)
	go pm.runDataProcTask(id, true)

	return nil
}

// runCollectProc выполняет сбор данных
func (pm *ProcManager) runCollectProc(ctx context.Context, rp *RunningProc) {
	defer rp.WG.Done()

	// Функция-триггер безопасно отправляет сигнал в канал при достижении порогового значения
	trigger := func() {
		select {
		case rp.Trigger <- struct{}{}:
		default:
			// Канал уже содержит сигнал, пропускаем
		}
	}

	var err error
	if rp.Cfg.Type == models.SrcNet {
		err = pm.collectFromNet(ctx, rp.Cfg, trigger)
	} else if rp.Cfg.Type == models.SrcDir || rp.Cfg.Type == models.SrcFile {
		err = pm.collectFromFileOrDir(ctx, rp.Cfg, trigger)
	}

	if err != nil && err != context.Canceled {
		logging.Log.Errorf("Ошибка в процессе %s: %v", rp.Cfg.ID, err)
		UpdProcStatus(rp.Cfg.ID, models.StatusError)
	}
}

func (pm *ProcManager) collectFromNet(ctx context.Context, p models.ProcCfg, trigger func()) error {
	return RunTsharkNet(ctx, p.IP, p.UDPPort, p.ID, trigger)
}

func (pm *ProcManager) collectFromFileOrDir(ctx context.Context, p models.ProcCfg, trigger func()) error {
	return RunTsharkFileOrDir(ctx, p, trigger)
}

// runTaskMgmtProc - Процесс управления задачами
func (pm *ProcManager) runTaskMgmtProc(ctx context.Context, rp *RunningProc) {
	defer rp.WG.Done()

	ticker := time.NewTicker(config.GlobalCfg.CacheDb.Timeout)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return // Корректно выходим при остановке
		case <-rp.Trigger: // Сигнал о достижении порога записей
			pm.spawnDataWorker(rp.Cfg.ID)
		case <-ticker.C: // Срабатывание таймера
			// Периодически очищаем SortedSet от устаревших данных
			//olderThan := time.Now().Add(-24 * time.Hour).UnixMilli()
			//_ = data.CleanupSortedSet(rp.Cfg.ID, olderThan)

			pm.spawnDataWorker(rp.Cfg.ID)
		}
	}
}

// spawnDataWorker запускает горутину обработки
func (pm *ProcManager) spawnDataWorker(srcID string) {
	// С одной структурой может работать только один процесс обработки данных одномоментно
	pm.srcLocksMu.Lock()
	if pm.srcLocks[srcID] {
		pm.srcLocksMu.Unlock()
		return // Воркер для этого источника уже работает
	}
	pm.srcLocks[srcID] = true
	pm.srcLocksMu.Unlock()

	// Одновременно может быть запущено до n таких процессов
	pm.workerMu.Lock()
	if pm.activeWorkers >= config.GlobalCfg.Generic.MaxProcs {
		pm.workerMu.Unlock()
		// Лимит превышен, снимаем блокировку источника, чтобы попробовать позже
		pm.srcLocksMu.Lock()
		pm.srcLocks[srcID] = false
		pm.srcLocksMu.Unlock()
		return
	}
	pm.activeWorkers++
	pm.workerMu.Unlock()

	go func() {
		defer func() {
			// Освобождаем лимиты и блокировки
			pm.workerMu.Lock()
			pm.activeWorkers--
			pm.workerMu.Unlock()

			pm.srcLocksMu.Lock()
			pm.srcLocks[srcID] = false
			pm.srcLocksMu.Unlock()
		}()

		pm.runDataProcTask(srcID, false)
	}()
}

// runDataProcTask - выполняет задачи фильтрации и переноса данных
func (pm *ProcManager) runDataProcTask(srcID string, isFinal bool) {
	count := int64(config.GlobalCfg.CacheDb.BatchSize)
	if isFinal {
		count = -1 // Забрать все элементы из очереди
	}

	for {
		entries, err := data.PopBatchFromList(srcID, count)
		if err != nil {
			logging.Log.Errorf("Ошибка обработки записи %s в кэширующей СУБД %v", srcID, err)
			return
		}
		if len(entries) == 0 {
			break // Очередь пуста, завершаем обработку
		}

		var pgEntries []data.DataEntry
		var zsEntries []data.ZSetEntry

		// Разделение и фильтрация
		for _, entryStr := range entries {
			key, value, ts, err := data.ImprovedParseEntry(entryStr)
			if err != nil {
				logging.Log.Errorf("Ошибка разбора записи %s: %v", entryStr, err)
				continue
			}

			// Фильтрация согласно белому и черному спискам
			if !data.IsAllowed(key, value) {
				continue
			}

			zsEntries = append(zsEntries, data.ZSetEntry{Key: key, Value: value, Timestamp: ts})
			pgEntries = append(pgEntries, data.DataEntry{
				Source:    srcID,
				Key:       key,
				Value:     value,
				Timestamp: ts,
			})
		}

		// Помещение в структуру sorted sets
		if len(zsEntries) > 0 {
			err = data.AddBatchToSortedSet(srcID, zsEntries)
			if err != nil {
				logging.Log.Errorf("Ошибка обработки записи в кэширующей СУБД %v", err)
			}
		}

		// Перемещение в PostgreSQL в пакетном режиме
		if len(pgEntries) > 0 {
			err = data.InsBatchInPg(srcID, pgEntries)
			if err != nil {
				logging.Log.Errorf("Ошибка вставки в СУБД PostgreSQL: %v", err)
			}
		}

		// Если это был финальный проход, выходим после одного раза
		if isFinal {
			break
		}
	}
}
