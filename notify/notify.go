package notify

import (
	"venera/logging"
	"venera/models"
)

// ProcessAlert обрабатывает сгенерированный алерт.
// Выводит его в систему логирования, Event Log или отправляет в GUI.
func ProcessAlert(event models.AlertEvent, severity string) {
	// Формируем детальное сообщение
	msg := event.Message
	if event.Key != "" || event.Value != "" {
		msg += " [source=" + event.Source + ", key=" + event.Key + ", value=" + event.Value + "]"
	}

	// Логирование согласно severity
	switch severity {
	case "critical", "error", "err":
		logging.Log.Error("ALERT: " + msg)
		// Запись критических алертов в Windows Event Log
		logEventToWindows(msg, "ERROR", 1000)
	case "warning", "warn":
		logging.Log.Warn("ALERT: " + msg)
	case "info":
		fallthrough
	default:
		logging.Log.Info("ALERT: " + msg)
	}

	// В будущем: здесь может быть вызов WebSocket для отправки алерта на фронтенд (React-UI)
}
