package metrics

import (
	"context"
	"time"

	"venera/logging"
	"venera/tray"
)

var (
	// callback на остановку всех процессов из менеджера (решение циклических зависимостей)
	stopAllProcesses func()
)

// SetStopAllCallback устанавливает функцию для остановки всех процессов
func SetStopAllCallback(fn func()) {
	stopAllProcesses = fn
}

// StartMonitor запускает фоновый процесс мониторинга системных порогов (п.17 ТЗ)
func StartMonitor(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second) // Проверка каждые 10 секунд
	defer ticker.Stop()

	warningSent := false

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// п.17.2: Остановка всех процессов с выводом сообщения при достижении критических порогов
			critical, msg := ProtectSystem()
			if critical {
				logging.Log.Error(msg)
				tray.ShowErrorNotification(msg)

				if stopAllProcesses != nil {
					logging.Log.Warn("Сработала защита от переполнения: автоматическая остановка всех процессов")
					stopAllProcesses()
				}
				continue
			}

			// п.17.1: Вывод сообщения пользователю при достижении N процентов свободного пространства
			if CheckDiskWarning() {
				if !warningSent {
					msg := "Внимание: мало свободного пространства на диске базы данных PostgreSQL!"
					logging.Log.Warn(msg)
					tray.ShowErrorNotification(msg)
					warningSent = true // Чтобы не спамить
				}
			} else {
				warningSent = false
			}
		}
	}
}
