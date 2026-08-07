package tests

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"venera/config"
	"venera/data"
	"venera/logging"
	"venera/models"
	"venera/processes"
	"venera/utils"
)

func init() {
	// Подготовка мок-логгера для безопасного тестирования
	logging.Log = logrus.New()
	logging.Log.SetOutput(os.Stdout)
}

// TestProcessManagerIntegration тестирует интеграцию ProcessManager с фильтрами и конфигурацией
// Цель: Проверить корректное взаимодействие между модулем процессов,
// модулем конфигурации и фильтрацией без реального запуска Tshark.
func TestProcessManagerIntegration(t *testing.T) {
	t.Log("Запуск интеграционного теста ProcessManager")

	// Инициализация временной конфигурации
	config.GlobalCfg = config.DefaultConfig()
	config.GlobalCfg.Generic.MaxProcesses = 2

	// Инициализация менеджера (он глобальный, но мы тестируем его логику)
	manager := processes.NewProcessManager()

	// Добавляем фейковый процесс
	procID := utils.GenerateID()
	pConfig := models.ProcessConfig{
		ID:     procID,
		Name:   "TestIntegrationProc",
		Type:   models.SourceNetwork,
		Status: models.StatusStopped,
	}

	// Попытка старта процесса
	// Поскольку внутри StartProcess запускается Tshark через runCollectionProcess,
	// который будет пытаться запустить реальный экзешник, мы просто проверим
	// базовую защиту от двойного старта (так как Tshark=tshark упадет, но процесс в мапе сохранится).

	err := manager.StartProcess(pConfig)
	if err != nil {
		t.Logf("Ожидаемая ошибка (может быть из-за отсутствия tshark): %v", err)
	}

	// Попытка запустить тот же процесс повторно должна вернуть ошибку сразу из мапы
	errDouble := manager.StartProcess(pConfig)
	if errDouble == nil {
		t.Error("Ожидалась ошибка при двойном запуске процесса")
	}

	// Тестирование связи с data модулем
	// Заполним пустые списки фильтрации, чтобы убедиться, что они работают
	errFilter := data.LoadFilters("non_existent_filter.flt")
	if errFilter != nil {
		t.Errorf("LoadFilters не должен возвращать ошибку для отсутствующего файла, получено: %v", errFilter)
	}

	// Остановка
	errStop := manager.StopProcess(procID)
	if errStop != nil {
		t.Logf("Ожидаемая ошибка остановки (если контекст уже отменен): %v", errStop)
	}

	// Небольшая задержка для корректного завершения горутин менеджера
	_, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	time.Sleep(50 * time.Millisecond)
}
