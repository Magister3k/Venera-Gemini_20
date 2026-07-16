package tray

import (
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/getlantern/systray"

	"venera/assets"
	"venera/config"
	"venera/logging"
	"venera/services"
)

var (
	onStart func()
	onStop  func()
)

// RunTray запускает приложение в виде иконки в системном трее Windows.
// Параметры startApp и stopApp - это функции запуска и остановки основной логики (веб-сервера и БД).
func RunTray(startApp func(), stopApp func()) {
	onStart = startApp
	onStop = stopApp

	// Запуск блокирующего цикла системного трея (systray.Run)
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetTitle("Venera")
	systray.SetTooltip("Venera Collector")

	// Установка иконки из модуля assets (п.21 ТЗ)
	if len(assets.IconBytes) > 0 {
		systray.SetIcon(assets.IconBytes)
	} else {
		logging.Log.Warn("Иконка трея не загружена (assets.IconBytes пуст)")
	}

	// Проверка прав администратора (п.16.1, 16.3 ТЗ)
	if !services.IsAdmin() {
		// ТЗ требует использовать systray для уведомления:
		// Добавляем неактивный пункт меню как всплывающее сообщение-предупреждение
		mAlert := systray.AddMenuItem("⚠️ ОШИБКА ДОСТУПА", "Отсутствуют права администратора")
		mAlert.Disable()
		mAlertMsg := systray.AddMenuItem("Перезапустите приложение с правами администратора", "")
		mAlertMsg.Disable()
		systray.AddSeparator()

		logging.Log.Warn("Приложение запущено без прав администратора (п.16 ТЗ)")
	}

	// Меню управления
	mOpenWeb := systray.AddMenuItem("Открыть веб-интерфейс", "Открыть панель управления Venera в браузере")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Выход", "Закрыть приложение Venera")

	// Асинхронно стартуем основную бизнес-логику и веб-сервер
	go onStart()

	// Обработчик событий кликов по меню и трею
	go func() {
		for {
			select {
			case <-mOpenWeb.ClickedCh:
				// п.10.1 ТЗ: Открытие веб-интерфейса по клику.
				// Библиотека systray не поддерживает нативный OnClick для самой иконки в Windows,
				// поэтому используется явный пункт меню (лучшая практика для systray).
				openWebInterface()

			case <-mQuit.ClickedCh:
				logging.Log.Info("Трей: запрошен выход из приложения")
				systray.Quit()
				return
			}
		}
	}()

	// В Windows иногда click по самой иконке может не срабатывать из-за особенностей systray.
	// Мы можем попробовать открыть веб-интерфейс, если systray поддерживает событие OnClick,
	// но в getlantern/systray это событие не всегда доступно/надежно без костылей.
}

func onExit() {
	logging.Log.Info("Трей закрывается...")
	onStop()
}

// openWebInterface открывает браузер по умолчанию на адресе веб-сервера (п.10.1 ТЗ).
func openWebInterface() {
	port := config.GlobalConfig.Generic.WebServerPort
	if port == 0 {
		port = 8080 // Fallback
	}

	url := fmt.Sprintf("http://127.0.0.1:%d", port)
	logging.Log.Infof("Открытие веб-интерфейса: %s", url)

	var err error
	switch runtime.GOOS {
	case "windows":
		// cmd /c start работает лучше, чем rundll32 url.dll,FileProtocolHandler в современных Windows
		err = exec.Command("cmd", "/c", "start", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}

	if err != nil {
		logging.Log.Errorf("Ошибка при открытии веб-интерфейса в браузере: %v", err)
	}
}

// ShowErrorNotification добавляет пункт меню с ошибкой для имитации уведомлений в трее (п.8.2 ТЗ)
// (Так как getlantern/systray не имеет встроенного метода ShowNotification).
func ShowErrorNotification(message string) {
	// Взаимодействие с GUI должно выполняться в основном потоке,
	// но systray thread-safe для AddMenuItem.
	mErr := systray.AddMenuItem("❌ Ошибка: "+message, "")
	mErr.Disable()

	// Авто-удаление сообщения через 10 секунд (опционально)
	go func() {
		time.Sleep(10 * time.Second)
		mErr.Hide() // Скрывает пункт из меню
	}()
}
