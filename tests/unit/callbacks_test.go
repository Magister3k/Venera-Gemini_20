package callbacks_test

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"venera/callbacks"
	"venera/logging"
)

func init() {
	// Инициализация мок-логгера для перехвата сообщений о панике
	logging.Log = logrus.New()
	logging.Log.SetOutput(os.Stdout)
}

func TestCallbacks(t *testing.T) {
	manager := callbacks.NewManager()

	var wg sync.WaitGroup
	wg.Add(1)

	// Тест успешного вызова
	manager.Subscribe("test_event", func(data interface{}) {
		defer wg.Done()
		val, ok := data.(string)
		if !ok || val != "hello" {
			t.Errorf("Ожидалось 'hello', получено: %v", data)
		}
	})

	manager.Emit("test_event", "hello")

	// Ожидаем завершения асинхронного коллбека с таймаутом
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Успех
	case <-time.After(2 * time.Second):
		t.Fatal("Таймаут ожидания коллбека")
	}

	// Тест паники
	manager.Subscribe("panic_event", func(data interface{}) {
		panic("Искусственная паника для теста")
	})

	// Приложение не должно упасть при вызове Emit
	manager.Emit("panic_event", nil)
	time.Sleep(100 * time.Millisecond) // Даем время горутине упасть и перехватиться

	// Тест очистки событий
	manager.ClearEvent("test_event")
	manager.Emit("test_event", "should_not_trigger")
}
