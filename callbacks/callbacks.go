package callbacks

import (
	"sync"
	"venera/logging"
)

// EventCallback определяет сигнатуру функции-коллбека.
type EventCallback func(data interface{})

// Manager управляет подписками и рассылкой событий.
// Данный модуль реализует паттерн Event Bus для предотвращения
// циклических зависимостей (import cycles) между модулями согласно п.18.1 ТЗ.
type Manager struct {
	mu        sync.RWMutex
	listeners map[string][]EventCallback
}

var (
	// GlobalManager - глобальная шина событий для использования всеми модулями
	GlobalManager = NewManager()
)

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

// ClearEvent удаляет всех слушателей для определенного события.
// Полезно при динамическом удалении процессов (п.1.4 ТЗ).
func (m *Manager) ClearEvent(event string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.listeners, event)
}

// Emit вызывает все коллбеки, подписанные на событие, с передачей данных.
func (m *Manager) Emit(event string, data interface{}) {
	m.mu.RLock()
	callbacks, ok := m.listeners[event]
	m.mu.RUnlock() // Отпускаем лок как можно раньше

	if ok {
		for _, cb := range callbacks {
			// Выполняем асинхронно, чтобы не блокировать вызывающего.
			// П.18.4 ТЗ: Безопасность при обработке паник.
			// Так как мы запускаем новую горутину, мы обязаны перехватить панику,
			// иначе она обрушит всё приложение.
			go func(callback EventCallback, eventData interface{}) {
				defer func() {
					if r := recover(); r != nil {
						logging.Log.Errorf("КРИТИЧЕСКАЯ ОШИБКА (Panic) в коллбеке события '%s': %v", event, r)
					}
				}()
				callback(eventData)
			}(cb, data)
		}
	}
}
