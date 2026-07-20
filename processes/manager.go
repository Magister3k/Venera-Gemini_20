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
	Manager = NewProcessManager()
)

// RunningProcess описывает состояние запущенного процесса
type RunningProcess struct {
	Config  models.ProcessConfig
	Cancel  context.CancelFunc
	Trigger chan struct{} // Канал для сигнализации о достижении порогового значения
	WG      sync.WaitGroup
}

type ProcessManager struct {
	mu            sync.Mutex
	activeProcs   map[string]*RunningProcess
	activeWorkers int
	workerMu      sync.Mutex

	// sourceLocks предотвращает одновременный запуск нескольких воркеров для одного источника
	sourceLocks   map[string]bool
	sourceLocksMu sync.Mutex
}

func NewProcessManager() *ProcessManager {
	return &ProcessManager{
		activeProcs: make(map[string]*RunningProcess),
		sourceLocks: make(map[string]bool),
	}
}

// StartProcess запускает процесс сбора данных
func (pm *ProcessManager) StartProcess(p models.ProcessConfig) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, exists := pm.activeProcs[p.ID]; exists {
		return fmt.Errorf("процесс %s уже запущен", p.ID)
	}

	ctx, cancel := context.WithCancel(context.Background())

	rp := &RunningProcess{
		Config:  p,
		Cancel:  cancel,
		Trigger: make(chan struct{}, 1), // Буферизованный канал на 1 сигнал для избежания deadlock
	}

	pm.activeProcs[p.ID] = rp

	UpdateProcessStatus(p.ID, models.StatusRunning)
	logging.Log.Infof("Запущен процесс: %s (%s)", p.Name, p.ID)

	rp.WG.Add(1)
	go pm.runCollectionProcess(ctx, rp)

	rp.WG.Add(1)
	go pm.runTaskManagementProcess(ctx, rp)

	return nil
}

// StopProcess останавливает процесс
func (pm *ProcessManager) StopProcess(id string) error {
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

	UpdateProcessStatus(id, models.StatusStopped)
	logging.Log.Infof("Остановлен процесс: %s", id)

	// Запускаем финальную обработку остатков в БД)
	go pm.runDataProcessingTask(id, true)

	return nil
}

// runCollectionProcess выполняет сбор данных
func (pm *ProcessManager) runCollectionProcess(ctx context.Context, rp *RunningProcess) {
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
	if rp.Config.Type == models.SourceNetwork {
		err = pm.collectFromNetwork(ctx, rp.Config, trigger)
	} else if rp.Config.Type == models.SourceFolder || rp.Config.Type == models.SourceFile {
		err = pm.collectFromFileOrFolder(ctx, rp.Config, trigger)
	}

	if err != nil && err != context.Canceled {
		logging.Log.Errorf("Ошибка в процессе %s: %v", rp.Config.ID, err)
		UpdateProcessStatus(rp.Config.ID, models.StatusError)
	}
}

func (pm *ProcessManager) collectFromNetwork(ctx context.Context, p models.ProcessConfig, trigger func()) error {
	return RunTsharkNetwork(ctx, p.IP, p.UDPPort, p.ID, trigger)
}

func (pm *ProcessManager) collectFromFileOrFolder(ctx context.Context, p models.ProcessConfig, trigger func()) error {
	return RunTsharkFileOrFolder(ctx, p, trigger)
}

// runTaskManagementProcess - Процесс управления задачами
func (pm *ProcessManager) runTaskManagementProcess(ctx context.Context, rp *RunningProcess) {
	defer rp.WG.Done()

	ticker := time.NewTicker(config.GlobalConfig.DragonflyDB.Timeout)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return // Корректно выходим при остановке
		case <-rp.Trigger: // Сигнал о достижении порога записей
			pm.spawnDataWorker(rp.Config.ID)
		case <-ticker.C: // Срабатывание таймера
			// Периодически очищаем SortedSet от устаревших данных
			olderThan := time.Now().Add(-24 * time.Hour).UnixMilli()
			_ = data.CleanupSortedSet(rp.Config.ID, olderThan)

			pm.spawnDataWorker(rp.Config.ID)
		}
	}
}

// spawnDataWorker запускает горутину обработки
func (pm *ProcessManager) spawnDataWorker(sourceID string) {
	// С одной структурой может работать только один процесс обработки данных одномоментно
	pm.sourceLocksMu.Lock()
	if pm.sourceLocks[sourceID] {
		pm.sourceLocksMu.Unlock()
		return // Воркер для этого источника уже работает
	}
	pm.sourceLocks[sourceID] = true
	pm.sourceLocksMu.Unlock()

	// Одновременно может быть запущено до n таких процессов
	pm.workerMu.Lock()
	if pm.activeWorkers >= config.GlobalConfig.Generic.MaxProcesses {
		pm.workerMu.Unlock()
		// Лимит превышен, снимаем блокировку источника, чтобы попробовать позже
		pm.sourceLocksMu.Lock()
		pm.sourceLocks[sourceID] = false
		pm.sourceLocksMu.Unlock()
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

			pm.sourceLocksMu.Lock()
			pm.sourceLocks[sourceID] = false
			pm.sourceLocksMu.Unlock()
		}()

		pm.runDataProcessingTask(sourceID, false)
	}()
}

// runDataProcessingTask - выполняет задачи фильтрации и переноса данных
func (pm *ProcessManager) runDataProcessingTask(sourceID string, isFinal bool) {
	count := int64(config.GlobalConfig.DragonflyDB.BatchSize)
	if isFinal {
		count = -1 // Забрать все элементы из очереди
	}

	for {
		entries, err := data.PopBatchFromList(sourceID, count)
		if err != nil {
			logging.Log.Errorf("Ошибка извлечения из list (%s): %v", sourceID, err)
			return
		}
		if len(entries) == 0 {
			break // Очередь пуста, завершаем обработку
		}

		var pgEntries []data.DataEntry
		var ssEntries []data.ZSetEntry

		// 5.5.2, 5.5.3: Разделение и фильтрация
		for _, entryStr := range entries {
			key, value, ts, err := data.ImprovedParseEntry(entryStr)
			if err != nil {
				logging.Log.Warnf("Ошибка парсинга записи: %v", err)
				continue
			}

			// Фильтрация согласно белому и черному спискам (data/filter.go)
			if !data.IsAllowed(key, value) {
				continue
			}

			ssEntries = append(ssEntries, data.ZSetEntry{Key: key, Value: value, Timestamp: ts})
			pgEntries = append(pgEntries, data.DataEntry{
				Source:    sourceID,
				Key:       key,
				Value:     value,
				Timestamp: ts,
			})
		}

		// 5.5.4: Помещение в структуру sorted sets
		if len(ssEntries) > 0 {
			err = data.AddBatchToSortedSet(sourceID, ssEntries)
			if err != nil {
				logging.Log.Errorf("Ошибка пакетного добавления в sorted set: %v", err)
			}
		}

		// 5.5.5: Перемещение в PostgreSQL в пакетном режиме
		if len(pgEntries) > 0 {
			err = data.InsertBatch(sourceID, pgEntries)
			if err != nil {
				logging.Log.Errorf("Ошибка вставки в PostgreSQL: %v", err)
			}
		}

		// Если это был финальный проход, выходим после одного раза
		if isFinal {
			break
		}
	}
}
