package test

import (
	"os"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"

	"venera/data"
	"venera/logging"
)

func init() {
	logging.Log = logrus.New()
	logging.Log.SetOutput(os.Stdout)
}

// TestConcurrentFilterAccess проверяет отсутствие Race Condition при одновременном
// чтении (IsAllowed) и записи (LoadFilters/Update) в мапу фильтрации.
func TestConcurrentFilterAccess(t *testing.T) {
	t.Log("Запуск стресс-теста: одновременный доступ к фильтрам (Race Condition Check)")

	var wg sync.WaitGroup

	// Горутина 1: Симулирует UI/Админа, который периодически обновляет фильтры
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			// Вызов загрузки (имитирует перезагрузку)
			_ = data.LoadFilters("dummy.flt")
		}
	}()

	// Горутины 2-11: Симулируют 10 рабочих процессов,
	// которые непрерывно читают фильтры для валидации данных Tshark.
	for w := 0; w < 10; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				_ = data.IsAllowed("network_ip", "192.168.0.1")
			}
		}()
	}

	wg.Wait()
	t.Log("Стресс-тест фильтров завершен успешно, паник не обнаружено.")
}
