package tray

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/getlantern/systray"

	"venera/assets"
	"venera/config"
	"venera/logging"
	"venera/utils"
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
	systray.SetTooltip("Система сбора идентификаторов в потоке пакетных данных")

	// Установка иконки из модуля assets
	if len(assets.IconBytes) > 0 {
		systray.SetIcon(assets.IconBytes)
	} else {
		logging.Log.Warn("Иконка трея не загружена")
	}

	cfg := config.GetCfg()

	// Скрываем консоль после полной загрузки tray, если было включено отображение
	if cfg.Generic.ShowConsoleOnStartup {
		utils.HideConsole()
	}

	// Проверка прав администратора
	if !utils.IsAdmin() {
		// Чтобы использовать systray для уведомления:
		// добавляем неактивный пункт меню как всплывающее сообщение-предупреждение
		mAlert := systray.AddMenuItem("⚠️ ОШИБКА ДОСТУПА", "Отсутствуют права администратора")
		mAlert.Disable()
		mAlertMsg := systray.AddMenuItem("Перезапустите приложение с правами администратора", "")
		mAlertMsg.Disable()
		systray.AddSeparator()

		logging.Log.Warn("Приложение запущено без прав Администратора")
		utils.ShowBalloonNotify("Venera", "Приложение запущено без прав Администратора. Некоторые функции могут быть недоступны.")
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
				// Открытие веб-интерфейса по клику.
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

// openWebInterface открывает веб-интерфейс в браузере по умолчанию.
func openWebInterface() {
	port := config.GlobalCfg.Generic.WebSrvPort
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
		err = fmt.Errorf("неподдерживаемая платформа")
	}

	if err != nil {
		logging.Log.Errorf("Ошибка при открытии веб-интерфейса в браузере: %v", err)
	}
}

// ShowErrorNotification добавляет пункт меню с ошибкой для имитации уведомлений в трее
/*func ShowErrorNotification(msg string) {
	// Взаимодействие с GUI должно выполняться в основном потоке,
	// но systray thread-safe для AddMenuItem.
	mErr := systray.AddMenuItem("❌ Ошибка: " + msg, "")
	mErr.Disable()

	// Авто-удаление сообщения через 10 секунд (опционально)
	go func() {
		time.Sleep(10 * time.Second)
		mErr.Hide() // Скрывает пункт из меню
	}()

	// Выводим системный Balloon
	utils.ShowBalloonNotify("Venera", msg)
}*/
