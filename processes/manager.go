package processes

import (
	"context"
	"fmt"
	"sync"
	"time"

	"venera/config"
	"venera/data"
	"venera/filter"
	"venera/logging"
	"venera/models"
)

var (
	Manager = NewProcessManager()
)

type RunningProcess struct {
	Config   models.ProcessConfig
	Cancel   context.CancelFunc
	Signal   *sync.Cond
	WG       sync.WaitGroup
	IsActive bool
}

type ProcessManager struct {
	mu            sync.Mutex
	activeProcs   map[string]*RunningProcess
	activeWorkers int
	workerMu      sync.Mutex
}

func NewProcessManager() *ProcessManager {
	return &ProcessManager{
		activeProcs: make(map[string]*RunningProcess),
	}
}

// StartProcess запускает процесс сбора данных ПВn
func (pm *ProcessManager) StartProcess(p models.ProcessConfig) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, exists := pm.activeProcs[p.ID]; exists {
		return fmt.Errorf("процесс %s уже запущен", p.ID)
	}

	ctx, cancel := context.WithCancel(context.Background())
	
	m := sync.Mutex{}
	cond := sync.NewCond(&m)

	rp := &RunningProcess{
		Config:   p,
		Cancel:   cancel,
		Signal:   cond,
		IsActive: true,
	}

	pm.activeProcs[p.ID] = rp

	UpdateProcessStatus(p.ID, "running")
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
	
	rp.IsActive = false
	rp.Cancel() // Отменяем контекст
	delete(pm.activeProcs, id)
	pm.mu.Unlock()

	// Ожидаем завершения горутин этого процесса
	rp.WG.Wait()

	UpdateProcessStatus(id, "stopped")
	logging.Log.Infof("Остановлен процесс: %s", id)

	// Запускаем финальную обработку остатков в БД
	go pm.runDataProcessingTask(id, true)

	return nil
}

// runCollectionProcess (ПВ1...ПВn)
func (pm *ProcessManager) runCollectionProcess(ctx context.Context, rp *RunningProcess) {
	defer rp.WG.Done()

	trigger := func() {
		rp.Signal.L.Lock()
		rp.Signal.Signal()
		rp.Signal.L.Unlock()
	}

	var err error
	if rp.Config.Type == models.SourceNetwork {
		err = pm.collectFromNetwork(ctx, rp.Config, trigger)
	} else if rp.Config.Type == models.SourceFolder || rp.Config.Type == models.SourceFile {
		err = pm.collectFromFileOrFolder(ctx, rp.Config, trigger)
	}

	if err != nil && err != context.Canceled {
		logging.Log.Errorf("Ошибка в процессе %s: %v", rp.Config.ID, err)
		UpdateProcessStatus(rp.Config.ID, "error")
	}
}

func (pm *ProcessManager) collectFromNetwork(ctx context.Context, p models.ProcessConfig, trigger func()) error {
	return RunTsharkNetwork(ctx, p.IP, p.UDPPort, p.ID, trigger)
}

func (pm *ProcessManager) collectFromFileOrFolder(ctx context.Context, p models.ProcessConfig, trigger func()) error {
	return RunTsharkFile(ctx, p.FilePath, p.FolderPath, p.ID, trigger)
}

// runTaskManagementProcess - Процесс управления задачами (п.4.2)
func (pm *ProcessManager) runTaskManagementProcess(ctx context.Context, rp *RunningProcess) {
	defer rp.WG.Done()

	ticker := time.NewTicker(config.GlobalConfig.DragonflyDB.Timeout)
	defer ticker.Stop()

	// Горутина для обработки сигнала cond, чтобы не блокировать select
	signalChan := make(chan struct{})
	go func() {
		for {
			rp.Signal.L.Lock()
			rp.Signal.Wait()
			rp.Signal.L.Unlock()
			
			if !rp.IsActive {
				return
			}
			
			select {
			case signalChan <- struct{}{}:
			default:
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			rp.Signal.L.Lock()
			rp.Signal.Broadcast() // Разбудить горутину ожидания, если она спит
			rp.Signal.L.Unlock()
			return
		case <-signalChan: // Сигнал о достижении порога (ПЗ)
			pm.spawnDataWorker(rp.Config.ID)
		case <-ticker.C: // Срабатывание таймера (ТПn)
			pm.spawnDataWorker(rp.Config.ID)
		}
	}
}

// spawnDataWorker запускает горутину обработки (ПОn), проверяя ограничения (Макс 20 процессов)
func (pm *ProcessManager) spawnDataWorker(sourceID string) {
	pm.workerMu.Lock()
	
	if pm.activeWorkers >= config.GlobalConfig.Generic.MaxProcesses {
		pm.workerMu.Unlock()
		// logging.Log.Warnf("Достигнут лимит рабочих процессов (%d). Ожидание...", config.GlobalConfig.Generic.MaxProcesses)
		return 
	}
	
	pm.activeWorkers++
	pm.workerMu.Unlock()

	go func() {
		defer func() {
			pm.workerMu.Lock()
			pm.activeWorkers--
			pm.workerMu.Unlock()
		}()
		
		pm.runDataProcessingTask(sourceID, false)
	}()
}

// runDataProcessingTask (ПОn) - выполняет задачи фильтрации и переноса (п.5.5)
func (pm *ProcessManager) runDataProcessingTask(sourceID string, isFinal bool) {
	count := int64(config.GlobalConfig.DragonflyDB.BatchSize)
	if isFinal {
		count = -1 // Забрать все
	}
	
	entries, err := data.PopBatchFromList(sourceID, count)
	if err != nil {
		logging.Log.Errorf("Ошибка извлечения из list (%s): %v", sourceID, err)
		return
	}
	if len(entries) == 0 {
		return
	}

	var pgEntries []data.DataEntry

	for _, entryStr := range entries {
		key, value, ts, err := data.ImprovedParseEntry(entryStr)
		if err != nil {
			logging.Log.Warnf("Ошибка парсинга записи: %v", err)
			continue
		}

		if !filter.IsKeyAllowed(key) {
			continue
		}
		if filter.IsValueBlocked(value) {
			continue
		}

		// Добавляем в Sorted Set для отображения
		err = data.AddToSortedSet(sourceID, key, value, ts)
		if err != nil {
			logging.Log.Errorf("Ошибка добавления в sorted set: %v", err)
			continue
		}
		
		pgEntries = append(pgEntries, data.DataEntry{
			Source:    sourceID,
			Key:       key,
			Value:     value,
			Timestamp: ts,
		})
	}

	// 5.5.5 Перемещение в PostgreSQL
	if len(pgEntries) > 0 {
		err = data.InsertBatch(sourceID, pgEntries)
		if err != nil {
			logging.Log.Errorf("Ошибка вставки в PostgreSQL: %v", err)
		} else {
			logging.Log.Infof("Успешно перенесено %d записей в PostgreSQL (источник: %s)", len(pgEntries), sourceID)
		}
	}
}