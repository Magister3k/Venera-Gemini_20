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

// StartMonitor запускает фоновый процесс мониторинга системных порогов
func StartMonitor(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second) // Проверка каждые 10 секунд
	defer ticker.Stop()

	warningSent := false

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Остановка всех процессов с выводом сообщения при достижении критических порогов
			// свободного места на диске с базой СУБД PostgreSQL и объема ОЗУ на ПК с программой
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

			// Вывод сообщения пользователю при достижении опасного порога свободного места на диске
			// с базой СУБД PostgreSQL
			if CheckDiskWarning() {
				if !warningSent {
					msg := "Внимание: мало свободного места на диске с базой СУБД PostgreSQL!"
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
