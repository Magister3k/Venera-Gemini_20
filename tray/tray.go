package tray

import (
	"venera/logging"

	"github.com/getlantern/systray"
)

var (
	onStart func()
	onStop  func()
)

// RunTray запускает приложение с иконкой в системном трее
func RunTray(startApp func(), stopApp func()) {
	onStart = startApp
	onStop = stopApp

	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetTitle("Venera")
	systray.SetTooltip("Venera Collector")
	
	// В реальном проекте тут нужно загрузить байты иконки, 
	// например через go:embed
	// systray.SetIcon(iconData)

	mStart := systray.AddMenuItem("Запустить сбор", "Запустить все процессы сбора")
	mStop := systray.AddMenuItem("Остановить сбор", "Остановить все процессы сбора")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Выход", "Закрыть приложение Venera")

	// Стартуем веб-сервер и БД при запуске трея
	go onStart()

	go func() {
		for {
			select {
			case <-mStart.ClickedCh:
				logging.Log.Info("Трей: запрошен ручной старт")
				// Здесь можно вызвать логику старта процессов (если не AutoStart)
			case <-mStop.ClickedCh:
				logging.Log.Info("Трей: запрошена ручная остановка")
				// Логика остановки процессов
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func onExit() {
	logging.Log.Info("Трей закрывается...")
	onStop()
}