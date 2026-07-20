package logging

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/windows/svc/eventlog"
)

// EventLogHook реализует интерфейс logrus.Hook для отправки
// критических ошибок в Windows Event Log.
type EventLogHook struct {
	sourceName string
	eventLog   *eventlog.Log
}

// NewEventLogHook создает новый хук для Windows Event Log
func NewEventLogHook(sourceName string) (*EventLogHook, error) {
	// Пытаемся подключиться к Event Log (Application)
	// Внимание: для создания нового источника (если его нет) требуются права администратора.
	// Если источник "VeneraApp" уже зарегистрирован (через манифест), он будет использован.
	elog, err := eventlog.Open(sourceName)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия Event Log источника '%s': %v", sourceName, err)
	}

	return &EventLogHook{
		sourceName: sourceName,
		eventLog:   elog,
	}, nil
}

// Levels возвращает уровни логирования, на которые будет
// реагировать хук "Отправка критических ошибок".
// Выбираем уровни Error, Fatal и Panic.
func (hook *EventLogHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.ErrorLevel,
		logrus.FatalLevel,
		logrus.PanicLevel,
	}
}

// Fire вызывается логгером при событии соответствующего уровня.
func (hook *EventLogHook) Fire(entry *logrus.Entry) error {
	msg, err := entry.String()
	if err != nil {
		msg = entry.Message
	}

	// Коды событий (Event IDs) могут быть жестко заданы или извлекаться из полей entry.Data
	// Для простоты используем базовые коды. В реальной системе можно передавать eventID через log.WithField("event_id", 1001).Error(...)
	eventID := uint32(1001) // Код по умолчанию для общих ошибок
	if idVal, ok := entry.Data["event_id"]; ok {
		if idNum, ok := idVal.(uint32); ok {
			eventID = idNum
		} else if idNum, ok := idVal.(int); ok {
			eventID = uint32(idNum)
		}
	}

	// Отправляем в Windows Event Log (Error)
	return hook.eventLog.Error(eventID, msg)
}

// Close закрывает соединение с Event Log
func (hook *EventLogHook) Close() {
	if hook.eventLog != nil {
		hook.eventLog.Close()
	}
}
