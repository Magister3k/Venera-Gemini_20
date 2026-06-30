package callbacks

import "sync"

// EventCallback определяет сигнатуру функции-коллбека.
type EventCallback func(data interface{})

// Manager управляет подписками и рассылкой событий.
type Manager struct {
	mu        sync.RWMutex
	listeners map[string][]EventCallback
}

// NewManager создает новый менеджер коллбеков.
func NewManager() *Manager {
	return &Manager{
		listeners: make(map[string][]EventCallback),
	}
}

// Subscribe добавляет слушателя на определенное событие.
func (m *Manager) Subscribe(event string, callback EventCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listeners[event] = append(m.listeners[event], callback)
}

// Emit вызывает все коллбеки, подписанные на событие, с передачей данных.
func (m *Manager) Emit(event string, data interface{}) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if callbacks, ok := m.listeners[event]; ok {
		for _, cb := range callbacks {
			// Выполняем асинхронно, чтобы не блокировать вызывающего
			go cb(data)
		}
	}
}
