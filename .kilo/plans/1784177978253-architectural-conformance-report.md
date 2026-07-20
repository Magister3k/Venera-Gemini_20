# Архитектурный план завершения разработки пользовательского интерфейса (React-UI & Fiber)

Этот документ представляет собой полную техническую спецификацию для завершения разработки веб-интерфейса Venera в соответствии с **разделами 9, 10 и 19 Технического задания**, утвержденной архитектурой и новыми требованиями по гибкой локализации интерфейса, справочным разделам и адаптивной верстке.

---

## 1. Архитектура разделения локального и удаленного интерфейсов (Local vs. Remote UI)

Для поддержки возможности выбора типа интерфейса на локальном ПК при обязательном использовании React-UI на удаленных рабочих местах (АРМ), в конфигурацию и бэкенд вносятся изменения.

### 1.1. Параметр в `config.toml` (`models/model_data.go`)
В структуру `GenericConfig` добавляется новый строковый параметр `local_ui_type`:
```go
type GenericConfig struct {
	// ... другие поля
	LocalUIType string `toml:"local_ui_type"` // "static" (jQuery в web/static) или "react" (React-UI в react-ui/dst)
}
```
*В `config.toml` по умолчанию выставляется `local_ui_type = "react"`.*

### 1.2. Middleware раздачи статики в зависимости от IP клиента (`web/server.go`)
В веб-сервере на базе Fiber настраивается динамическое обслуживание статических файлов:
```go
// Определение локального адреса
func isLocalRequest(ip string) bool {
	return ip == "127.0.0.1" || ip == "::1" || ip == "localhost"
}

// В методе StartWebServer():
app.Use(func(c *fiber.Ctx) error {
	cfg := config.GetConfig()
	clientIP := c.IP()
	
	// По умолчанию для удаленных АРМ всегда отдаем React-UI
	staticDir := "./react-ui/dst"
	
	// Если запрос локальный и в конфиге выбран тип "static" (п.11 ТЗ)
	if isLocalRequest(clientIP) && cfg.Generic.LocalUIType == "static" {
		staticDir = "./web/static"
	}
	
	c.Locals("static_dir", staticDir)
	return c.Next()
})

// Кастомный обработчик статики, считывающий выбранную директорию
app.Get("/*", func(c *fiber.Ctx) error {
	staticDir := c.Locals("static_dir").(string)
	
	// Проверяем наличие файла в выбранной директории
	filePath := filepath.Join(staticDir, c.Path())
	if _, err := os.Stat(filePath); err != nil {
		// Для SPA отдаем index.html той же выбранной директории
		return c.SendFile(filepath.Join(staticDir, "index.html"))
	}
	
	return c.SendFile(filePath)
})
```

---

## 2. Справочные разделы: "Помощь" и "О программе"

Для предоставления справочной информации пользователю создаются новые эндпоинты бэкенда и соответствующие виды (views) на фронтенде.

### 2.1. Спецификация REST API (`web/handlers.go`)
- **GET `/api/help`**: Возвращает структурированное руководство пользователя в формате JSON.
  ```json
  {
    "sections": [
      {
        "title": "Введение",
        "content": "Venera — это высокопроизводительная система сбора и фильтрации сетевых идентификаторов в реальном времени."
      },
      {
        "title": "Управление процессами",
        "content": "Для запуска сбора создайте процесс в панели 'Процессы'. Укажите тип источника (сетевой интерфейс, папка или файл PCAP) и нажмите 'Старт'."
      },
      {
        "title": "Фильтрация и контроль",
        "content": "Фильтрация данных производится по спискам generic.flt (белый список ключей и черный список значений). Контролируемые значения задаются в generic.ctr."
      }
    ]
  }
  ```
- **GET `/api/about`**: Возвращает информацию о программном обеспечении.
  ```json
  {
    "app_name": "Venera Collector",
    "version": "1.0.0",
    "developer": "Venera Team",
    "build_time": "2026-07-21",
    "license": "Проприетарная (Для внутреннего использования)",
    "copyright": "© 2026 Venera Team. Все права защищены."
  }
  ```

### 2.2. Фронтенд компоненты (`react-ui/src/components/`)
* **`Help.jsx`**: Отображает руководство пользователя в виде удобного аккордеона (Accordion) с возможностью быстрого поиска по разделам справки.
* **`About.jsx`**: Презентационная карточка с логотипом программы (иконка), версией, лицензионным соглашением и системной информацией о сборке.

---

## 3. Адаптивная навигационная панель с автоскрытием и фиксацией (React-UI)

Навигационное меню React-приложения должно поддерживать режим автоматического сворачивания для экономии рабочего пространства на АРМ операторов с возможностью жесткого закрепления (pin).

### 3.1. Логика компонента (`react-ui/src/App.jsx`)
Компонент `App` управляет состояниями свертки:
* **`isPinned`**: Состояние фиксации панели. Сохраняется в `localStorage` (ключ `venera-nav-pinned`).
* **`isHovered`**: Состояние наведения курсора мыши на панель.

```jsx
const [isPinned, setIsPinned] = useState(() => localStorage.getItem('venera-nav-pinned') === 'true');
const [isHovered, setIsHovered] = useState(false);

const togglePin = () => {
    const nextState = !isPinned;
    setIsPinned(nextState);
    localStorage.setItem('venera-nav-pinned', String(nextState));
};

// Панель считается развернутой, если она закреплена ИЛИ на нее наведен курсор
const isExpanded = isPinned || isHovered;
```

### 3.2. Верстка и CSS стили (`react-ui/src/index.css`)
Для плавной анимации скрытия используется CSS transition:
```css
nav {
  width: 50px; /* Свернутое состояние (показываем только иконки) */
  background-color: var(--nav-bg);
  border-right: 1px solid var(--border-color);
  padding: 15px 10px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  transition: width 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  overflow: hidden;
  white-space: nowrap;
}

nav.expanded {
  width: 250px; /* Развернутое состояние */
}

/* Кнопка закрепления (Pin) вверху панели */
.pin-button {
  background: none;
  border: none;
  color: var(--text-color);
  cursor: pointer;
  align-self: flex-end;
  font-size: 1.2rem;
  padding: 5px;
  transition: transform 0.2s;
}

.pin-button.pinned {
  transform: rotate(-45deg); /* Поворот иконки кнопки при закреплении */
  color: #007bff;
}

/* Скрытие текстовых меток меню при свернутой панели */
nav a span {
  opacity: 0;
  transition: opacity 0.2s;
  pointer-events: none;
}

nav.expanded a span {
  opacity: 1;
  pointer-events: auto;
}
```

---

## 4. Архитектура броадкастинга логов в реальном времени (Real-Time Logs)

Для реализации стриминга логов через WebSocket по эндпоинту `/ws/logs` (п.10.8 ТЗ) будет использоваться спроектированная шина событий в памяти (`LogHub`), интегрированная в пакет `logging` в качестве хука `logrus`.

### 4.1. Структура `LogHub` (`logging/hub.go`)
```go
package logging

import (
	"sync"
	"github.com/sirupsen/logrus"
)

type LogHub struct {
	mu          sync.Mutex
	subscribers map[chan string]bool
}

var Hub = &LogHub{
	subscribers: make(map[chan string]bool),
}

func (h *LogHub) Subscribe() chan string {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch := make(chan string, 100) // Буфер для предотвращения блокировок
	h.subscribers[ch] = true
	return ch
}

func (h *LogHub) Unsubscribe(ch chan string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.subscribers, ch)
	close(ch)
}

func (h *LogHub) Broadcast(msg string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subscribers {
		select {
		case ch <- msg:
		default:
			// Сбрасываем сообщение, если буфер переполнен (медленный клиент)
		}
	}
}

// Logrus Hook для перехвата логов
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
```

---

## 5. Спецификация остальных API-эндпоинтов бэкенда (Go + Fiber)

### 5.1. Групповые операции процессов (`web/handlers.go`)
- **POST `/api/processes/actions`**: Принимает query-параметр `?action=start|stop|delete` и применяется действие ко всем процессам одновременно (п.10.4 ТЗ).

### 5.2. Проверка подключений (`web/handlers.go`)
- **POST `/api/config/test-postgres`**: Временно открывает пул pgx с переданными параметрами и возвращает статус подключения.
- **POST `/api/config/test-dragonfly`**: Аналогично проверяет пинг к DragonflyDB.

### 5.3. Потоковый экспорт PostgreSQL в XLSX (`web/handlers.go`)
Для предотвращения утечек памяти и OOM (Out of Memory) при выгрузке больших объемов данных (п.10.6 ТЗ) будет использоваться потоковый писатель библиотеки `excelize/v2` (`StreamWriter`).
- **GET `/api/db/export/xlsx`**:
  - Инициализирует пустой Excel файл: `f := excelize.NewFile()`.
  - Открывает `StreamWriter`: `sw, _ := f.NewStreamWriter("Sheet1")`.
  - Выполняет запрос к PostgreSQL через курсор `pgx.Rows` для последовательного чтения записей без удержания их в памяти.
  - Построчно записывает данные в Excel: `sw.SetRow(...)`.
  - Закрывает `StreamWriter` и отправляет сгенерированный файл клиенту как поток байт.

### 5.4. Интеграция диагностики (`web/handlers.go`)
- **POST `/api/diagnose/run`**: Запускает `diagnose.RunDiagnosis()` и возвращает результаты проверок в JSON (п.10.9 ТЗ).
- **GET `/api/diagnose/download/pdf`**: Генерирует PDF-отчет на русском языке с помощью `diagnose.ExportReportPDF()` и возвращает файл как вложение.
- **GET `/api/diagnose/download/archive`**: Запускает `diagnose.CreateArchiveGZ()` и отдает архив `.tar.gz` (п.9.1.17 ТЗ).

---

## 6. План валидации и предотвращения Race Conditions

1. **Тестирование переключения интерфейсов**:
   - Запустить Venera локально, выставить `local_ui_type = "static"`. Проверить, что при обращении с `127.0.0.1:8080` отдаются файлы из `web/static`, а при обращении с любого внешнего IP — открывается React-UI.
2. **Проверка утечек памяти в WebSocket**:
   - Эмулировать частое переподключение клиентов (клиент открывает и закрывает сокет логов 100 раз подряд). Убедиться, что в `LogHub` количество подписчиков возвращается к исходному состоянию.
3. **Объемный тест (XLSX)**:
   - Проверить экспорт в Excel выборки объемом 10 000 строк. Измерить потребление RAM процессом `venera.exe`.
4. **Проверка сохранения состояния панели навигации**:
   - Закрепить панель кликом по иконке пина, перезагрузить страницу. Убедиться, что панель осталась закрепленной (`localStorage` работает).
