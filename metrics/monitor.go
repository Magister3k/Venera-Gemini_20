package metrics

import (
	"context"
	"time"

	"venera/logging"
	"venera/processes"
	"venera/utils"
)

var (
	// callback на остановку всех процессов из менеджера (решение циклических зависимостей)
	stopAllProcs func()
)

// SetStopAllCallback устанавливает функцию для остановки всех процессов
func SetStopAllCallback(fn func()) {
	stopAllProcs = fn
}

// StartMonitor запускает фоновый процесс мониторинга системных порогов
func StartMonitor(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second) // Проверка каждые 10 секунд
	defer ticker.Stop()

	warnSent := false

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Остановка всех процессов с выводом сообщения при достижении критических порогов
			// свободного места на диске с итоговой БД и объема ОЗУ на ПК с программой
			critical, msg := ProtectSystem()
			if critical {
				logging.Log.Error(msg)
				utils.ShowBalloonNotification("Venera", msg)

				if processes.GetAlLProcs() != nil {
					logging.Log.Warn("Сработала защита от переполнения: автоматическая остановка всех процессов")
					StopAllProcs()
				}
				continue
			}

			// Вывод сообщения пользователю при достижении опасного порога свободного места на диске
			// с итоговой базой в СУБД PostgreSQL
			if CheckDiskWarn() {
				if !warnSent {
					msg := "Внимание: мало свободного места на диске с итоговой базой в СУБД PostgreSQL!"
					logging.Log.Warn(msg)
					utils.ShowBalloonNotification("Venera", msg)
					warnSent = true // Чтобы не спамить
				}
			} else {
				warnSent = false
			}
		}
	}
}
