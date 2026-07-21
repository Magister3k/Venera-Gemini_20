# Архитектурный план завершения разработки пользовательского интерфейса (React-UI & Fiber)

Этот документ представляет собой полную техническую спецификацию для завершения разработки веб-интерфейса Venera в соответствии с **разделами 9, 10 и 19 Технического задания**, утвержденной архитектурой и новыми требованиями по гибкой локализации интерфейса, справочным разделам, адаптивной верстке и управлению отображением консоли.

---

## 1. Архитектура управления консолью и Balloon-уведомлениями (Windows)

Для повышения удобства эксплуатации и гибкости настройки запуска приложения в режиме трея, добавляется параметр управления видимостью консоли при старте с механизмом автоматического скрытия и перевода уведомлений во всплывающие окна Windows.

### 1.1. Параметры в `config.toml` (`models/model_data.go` и `config/cfg_loader.go`)
В структуру `GenericConfig` добавляется новый логический параметр `show_console_on_startup`:
```go
type GenericConfig struct {
	// ... другие поля
	ShowConsoleOnStartup bool `toml:"show_console_on_startup"` // true - показывать консоль при старте, false - сразу скрывать
}
```
*В `config.toml` по умолчанию выставляется `show_console_on_startup = true`.*

### 1.2. Утилиты скрытия консоли (`utils/console.go`)
Через WinAPI `syscall` реализуется управление отображением окна консоли:
```go
package utils

import "syscall"

var (
	user32           = syscall.NewLazyDLL("user32.dll")
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procGetConsole   = kernel32.NewProc("GetConsoleWindow")
	procShowWindow   = user32.NewProc("ShowWindow")
)

const (
	SW_HIDE = 0
	SW_SHOW = 5
)

// HideConsole скрывает окно консоли текущего процесса в Windows
func HideConsole() {
	hwnd, _, _ := procGetConsole.Call()
	if hwnd != 0 {
		_, _, _ = procShowWindow.Call(hwnd, uintptr(SW_HIDE))
	}
}

// ShowConsole отображает окно консоли текущего процесса в Windows
func ShowConsole() {
	hwnd, _, _ := procGetConsole.Call()
	if hwnd != 0 {
		_, _, _ = procShowWindow.Call(hwnd, uintptr(SW_SHOW))
	}
}
```

### 1.3. Реализация всплывающих Balloon-уведомлений (`utils/notification.go`)
Для полноценного вывода уведомлений на Windows без использования сторонних CGO библиотек используется асинхронный вызов PowerShell-скрипта через `exec.Command`:
```go
package utils

import (
	"fmt"
	"os/exec"
)

// ShowBalloonNotification отображает системное всплывающее уведомление Windows (Balloon)
func ShowBalloonNotification(title, message string) {
	psCmd := fmt.Sprintf(`
		[void][System.Reflection.Assembly]::LoadWithPartialName("System.Windows.Forms");
		$notification = New-Object System.Windows.Forms.NotifyIcon;
		$notification.Icon = [System.Drawing.SystemIcons]::Information;
		$notification.BalloonTipIcon = "Info";
		$notification.BalloonTipTitle = "%s";
		$notification.BalloonTipText = "%s";
		$notification.Visible = $true;
		$notification.ShowBalloonTip(5000);
	`, title, message)

	cmd := exec.Command("powershell", "-NoProfile", "-Command", psCmd)
	_ = cmd.Start() // Асинхронный запуск
}
```

### 1.4. Интеграция в жизненный цикл запуска (`main.go` и `tray/tray.go`)
- **В `main.go`**: При старте, если `cfg.Generic.Mode == "tray"` и `cfg.Generic.ShowConsoleOnStartup == false`, сразу вызывается `utils.HideConsole()`.
- **В `tray/tray.go` (в методе `onReady`)**:
  - Если `cfg.Generic.ShowConsoleOnStartup == true`, по окончании инициализации трея вызывается `utils.HideConsole()`.
  - Все критические сообщения (ошибки инициализации, алерты СУБД) транслируются во всплывающий баллон через `utils.ShowBalloonNotification()`.
  ```go
  // Пример в onReady() при отсутствии прав администратора:
  if !utils.IsAdmin() {
      utils.ShowBalloonNotification("Ошибка доступа", "Перезапустите приложение с правами администратора")
  }
  ```

---

## 2. Архитектура разделения локального и удаленного интерфейсов (Local vs. Remote UI)

Для поддержки возможности выбора типа интерфейса на локальном ПК при обязательном использовании React-UI на удаленных рабочих местах (АРМ), в конфигурацию и бэкенд вносятся изменения.

### 2.1. Параметр в `config.toml` (`models/model_data.go`)
В структуру `GenericConfig` добавляется новый строковый параметр `local_ui_type`:
```go
type GenericConfig struct {
	// ... другие поля
	LocalUIType string `toml:"local_ui_type"` // "static" (jQuery в web/static) или "react" (React-UI в react-ui/dst)
}
```

### 2.2. Middleware раздачи статики в зависимости от IP клиента (`web/server.go`)
В веб-сервере на базе Fiber настраивается динамическое обслуживание статических файлов:
```go
func isLocalRequest(ip string) bool {
	return ip == "127.0.0.1" || ip == "::1" || ip == "localhost"
}

// В методе StartWebServer():
app.Use(func(c *fiber.Ctx) error {
	cfg := config.GetConfig()
	clientIP := c.IP()
	staticDir := "./react-ui/dst"
	
	if isLocalRequest(clientIP) && cfg.Generic.LocalUIType == "static" {
		staticDir = "./web/static"
	}
	
	c.Locals("static_dir", staticDir)
	return c.Next()
})
```

---

## 3. Справочные разделы: "Помощь" и "О программе"

Для предоставления справочной информации пользователю создаются новые эндпоинты бэкенда и соответствующие виды (views) на фронтенде.

### 3.1. Спецификация REST API (`web/handlers.go`)
- **GET `/api/help`**: Возвращает структурированное руководство пользователя в формате JSON.
- **GET `/api/about`**: Возвращает информацию о программном обеспечении (версия, сборка, лицензия).

### 3.2. Фронтенд компоненты (`react-ui/src/components/`)
* **`Help.jsx`**: Отображает руководство пользователя в виде удобного аккордеона (Accordion) с возможностью быстрого поиска по разделам справки.
* **`About.jsx`**: Презентационная карточка с логотипом программы (иконка), версией, лицензионным соглашением и системной информацией о сборке.

---

## 4. Адаптивная навигационная панель с автоскрытием и фиксацией (React-UI)

Навигационное меню React-приложения должно поддерживать режим автоматического сворачивания для экономии рабочего пространства на АРМ операторов с возможностью жесткого закрепления (pin).

### 4.1. Логика компонента (`react-ui/src/App.jsx`)
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

---

## 5. Архитектура броадкастинга логов в реальном времени (Real-Time Logs)

Для реализации стриминга логов через WebSocket по эндпоинту `/ws/logs` (п.10.8 ТЗ) будет использоваться спроектированная шина событий в памяти (`LogHub`), интегрированная в пакет `logging` в качестве хука `logrus`.

### 5.1. Структура `LogHub` (`logging/hub.go`)
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

## 6. Спецификация остальных API-эндпоинтов бэкенда (Go + Fiber)

### 6.1. Групповые операции процессов (`web/handlers.go`)
- **POST `/api/processes/actions`**: Принимает query-параметр `?action=start|stop|delete` и применяется действие ко всем процессам одновременно (п.10.4 ТЗ).

### 6.2. Проверка подключений (`web/handlers.go`)
- **POST `/api/config/test-postgres`**: Временно открывает пул pgx с переданными параметрами и возвращает статус подключения.
- **POST `/api/config/test-dragonfly`**: Аналогично проверяет пинг к DragonflyDB.

### 6.3. Потоковый экспорт PostgreSQL в XLSX (`web/handlers.go`)
Для предотвращения утечек памяти и OOM (Out of Memory) при выгрузке больших объемов данных (п.10.6 ТЗ) будет использоваться потоковый писатель библиотеки `excelize/v2` (`StreamWriter`).
- **GET `/api/db/export/xlsx`**:
  - Инициализирует пустой Excel файл: `f := excelize.NewFile()`.
  - Открывает `StreamWriter`: `sw, _ := f.NewStreamWriter("Sheet1")`.
  - Выполняет запрос к PostgreSQL через курсор `pgx.Rows` для последовательного чтения записей без удержания их в памяти.
  - Построчно записывает данные в Excel: `sw.SetRow(...)`.
  - Закрывает `StreamWriter` и отправляет сгенерированный файл клиенту как поток байт.

### 6.4. Интеграция диагностики (`web/handlers.go`)
- **POST `/api/diagnose/run`**: Запускает `diagnose.RunDiagnosis()` и возвращает результаты проверок в JSON (п.10.9 ТЗ).
- **GET `/api/diagnose/download/pdf`**: Генерирует PDF-отчет на русском языке с помощью `diagnose.ExportReportPDF()` и возвращает файл как вложение.
- **GET `/api/diagnose/download/archive`**: Запускает `diagnose.CreateArchiveGZ()` и отдает архив `.tar.gz` (п.9.1.17 ТЗ).

### 6.5. Реализация веб-сокетов (`web/websocket.go`)
- **WS `/ws/logs`**: Создает подписку `ch := logging.Hub.Subscribe()`. В цикле читает из канала и отправляет в сокет через `c.WriteMessage()`. При выходе вызывает `logging.Hub.Unsubscribe(ch)`.
- **WS `/ws/db`**: Позволяет искать и фильтровать данные в реальном времени.

---

## 7. План валидации и предотвращения Race Conditions

1. **Тестирование переключения интерфейсов**:
   - Запустить Venera локально, выставить `local_ui_type = "static"`. Проверить, что при обращении с `127.0.0.1:8080` отдаются файлы из `web/static`, а при обращении с любого внешнего IP — открывается React-UI.
2. **Проверка утечек памяти в WebSocket**:
   - Эмулировать частое переподключение клиентов (клиент открывает и закрывает сокет логов 100 раз подряд). Убедиться, что в `LogHub` количество подписчиков возвращается к исходному состоянию.
3. **Объемный тест (XLSX)**:
   - Проверить экспорт в Excel выборки объемом 10 000 строк. Измерить потребление RAM процессом `venera.exe`.
4. **Проверка сохранения состояния панели навигации**:
   - Закрепить панель кликом по иконке пина, перезагрузить страницу. Убедиться, что панель осталась закрепленной (`localStorage` работает).
5. **Проверка скрытия консоли**:
   - Проверить запуск приложения с `show_console_on_startup = true`. Консоль должна открываться, а затем скрываться сразу после инициализации `systray`. Проверить, что ошибки выводятся во всплывающих системных Balloon-уведомлениях Windows.
6. **Проверка работы Balloon-уведомлений**:
   - Инициировать предупреждение о диске (выставить критический порог диска в 99%). Убедиться, что всплывает красивое уведомление Windows.
