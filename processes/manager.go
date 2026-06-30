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

type ProcessManager struct {
	mu           sync.Mutex
	activeProcs  map[string]context.CancelFunc
	channels     map[string]chan struct{} // КАН1...КАНn
	wg           sync.WaitGroup
	activeWorkers int // Текущее количество процессов обработки (ПО1...ПОn)
	workerMu      sync.Mutex
}

func NewProcessManager() *ProcessManager {
	return &ProcessManager{
		activeProcs: make(map[string]context.CancelFunc),
		channels:    make(map[string]chan struct{}),
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
	pm.activeProcs[p.ID] = cancel
	
	// Создаем канал для сигнализации (КАНn)
	pm.channels[p.ID] = make(chan struct{}, 1)

	UpdateProcessStatus(p.ID, "running")
	logging.Log.Infof("Запущен процесс: %s (%s)", p.Name, p.ID)

	pm.wg.Add(1)
	go pm.runCollectionProcess(ctx, p)
	
	pm.wg.Add(1)
	go pm.runTaskManagementProcess(ctx, p.ID)

	return nil
}

// StopProcess останавливает процесс
func (pm *ProcessManager) StopProcess(id string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	cancel, exists := pm.activeProcs[id]
	if !exists {
		return fmt.Errorf("процесс %s не запущен", id)
	}

	cancel() // Отменяем контекст
	delete(pm.activeProcs, id)
	
	if ch, ok := pm.channels[id]; ok {
		close(ch)
		delete(pm.channels, id)
	}

	UpdateProcessStatus(id, "stopped")
	logging.Log.Infof("Остановлен процесс: %s", id)

	// Запускаем финальную обработку остатков в БД
	go pm.runDataProcessingTask(id, true)

	return nil
}

// runCollectionProcess (ПВ1...ПВn)
func (pm *ProcessManager) runCollectionProcess(ctx context.Context, p models.ProcessConfig) {
	defer pm.wg.Done()

	var err error
	if p.Type == models.SourceNetwork {
		err = pm.collectFromNetwork(ctx, p)
	} else if p.Type == models.SourceFolder || p.Type == models.SourceFile {
		err = pm.collectFromFileOrFolder(ctx, p)
	}

	if err != nil && err != context.Canceled {
		logging.Log.Errorf("Ошибка в процессе %s: %v", p.ID, err)
		UpdateProcessStatus(p.ID, "error")
	}
}

func (pm *ProcessManager) collectFromNetwork(ctx context.Context, p models.ProcessConfig) error {
	// Здесь должен быть вызов Tshark для сети
	// Эмуляция для примера
	return RunTsharkNetwork(ctx, p.IP, p.UDPPort, p.ID, pm.channels[p.ID])
}

func (pm *ProcessManager) collectFromFileOrFolder(ctx context.Context, p models.ProcessConfig) error {
	// Здесь должен быть вызов Tshark для файлов
	return RunTsharkFile(ctx, p.FilePath, p.FolderPath, p.ID, pm.channels[p.ID])
}

// runTaskManagementProcess - Процесс управления задачами (п.4.2)
func (pm *ProcessManager) runTaskManagementProcess(ctx context.Context, sourceID string) {
	defer pm.wg.Done()

	ticker := time.NewTicker(config.GlobalConfig.DragonflyDB.Timeout)
	defer ticker.Stop()

	pm.mu.Lock()
	ch := pm.channels[sourceID]
	pm.mu.Unlock()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ch: // Сигнал о достижении порога (ПЗ)
			pm.spawnDataWorker(sourceID)
		case <-ticker.C: // Срабатывание таймера (ТПn)
			pm.spawnDataWorker(sourceID)
		}
	}
}

// spawnDataWorker запускает горутину обработки (ПОn), проверяя ограничения (Макс 20 процессов)
func (pm *ProcessManager) spawnDataWorker(sourceID string) {
	pm.workerMu.Lock()
	
	// Проверяем лимит (п.1.5)
	if pm.activeWorkers >= config.GlobalConfig.Generic.MaxProcesses {
		pm.workerMu.Unlock()
		logging.Log.Warnf("Достигнут лимит рабочих процессов (%d). Ожидание...", config.GlobalConfig.Generic.MaxProcesses)
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
	// 5.5.1. Получение записей из структуры list с удалением
	count := int64(config.GlobalConfig.DragonflyDB.BatchSize)
	if isFinal {
		count = -1 // Забрать все
	}
	
	// В реальной реализации нужно забирать батчами
	entries, err := data.PopBatchFromList(sourceID, count)
	if err != nil {
		logging.Log.Errorf("Ошибка извлечения из list (%s): %v", sourceID, err)
		return
	}
	if len(entries) == 0 {
		return
	}

	// 5.5.2, 5.5.3, 5.5.4
	var pgEntries []data.DataEntry

	for _, entryStr := range entries {
		key, value, ts, err := data.ImprovedParseEntry(entryStr)
		if err != nil {
			logging.Log.Warnf("Ошибка парсинга записи: %v", err)
			continue
		}

		// Фильтрация (п.5.5.3)
		if !data.IsKeyAllowed(key) {
			continue // Не в белом списке
		}
		if data.IsValueBlocked(value) {
			continue // В черном списке
		}

		// Проверка на алерт (п.1.10) - здесь можно вызывать notify.CheckAlert

		// 5.5.4 Помещение в sorted sets
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

	// Извлекаем из SortedSet и очищаем (в данном флоу мы можем сразу переносить pgEntries)
	// Для строгого следования п.5.5.5: Перемещение всех записей из sorted sets в PostgreSQL
	ssData, err := data.GetAndClearSortedSet(sourceID)
	if err != nil {
		logging.Log.Errorf("Ошибка очистки sorted set: %v", err)
	} else {
		// Обновляем pgEntries на основе ssData, если нужно строго из SS брать
		// В этой упрощенной модели мы уже имеем pgEntries с ts
		_ = ssData 
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
	
	// 5.5.6. Закрытие процесса обработки (происходит автоматически при выходе из функции)
}
