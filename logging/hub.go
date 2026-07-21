package logging

import (
	"sync"

	"github.com/sirupsen/logrus"
)

// LogHub управляет WebSocket-подписчиками для трансляции логов в реальном времени
type LogHub struct {
	mu          sync.Mutex
	subscribers map[chan string]bool
}

// Hub - глобальный броадкастер логов
var Hub = &LogHub{
	subscribers: make(map[chan string]bool),
}

// Subscribe создает новый канал-подписчик
func (h *LogHub) Subscribe() chan string {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch := make(chan string, 200) // Буфер для медленных сокетов
	h.subscribers[ch] = true
	return ch
}

// Unsubscribe безопасно закрывает и удаляет подписчика
func (h *LogHub) Unsubscribe(ch chan string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, exists := h.subscribers[ch]; exists {
		delete(h.subscribers, ch)
		close(ch)
	}
}

// Broadcast отправляет сообщение всем активным клиентам
func (h *LogHub) Broadcast(msg string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subscribers {
		select {
		case ch <- msg:
		default:
			// Сбрасываем сообщение, если буфер клиента переполнен, чтобы не тормозить систему
		}
	}
}

// WsLogHook перехватывает записи logrus и отправляет их в Hub
type WsLogHook struct{}

func (h *WsLogHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *WsLogHook) Fire(entry *logrus.Entry) error {
	line, err := entry.String()
	if err == nil {
		Hub.Broadcast(line)
	}
	return nil
}
