package tray

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/getlantern/systray"
	"venera/config"
	"venera/logging"
)

// IconData должен быть заполнен байтами иконки
var IconData []byte

// RunTray запускает приложение в системном трее
func RunTray(startApp func(), stopApp func()) {
	systray.Run(func() {
		onReady(startApp)
	}, stopApp)
}

func onReady(startApp func()) {
	if len(IconData) > 0 {
		systray.SetIcon(IconData)
	}
	systray.SetTitle("Venera")
	systray.SetTooltip("Система сбора идентификаторов Venera")

	mOpen := systray.AddMenuItem("Открыть интерфейс", "Открыть веб-интерфейс управления")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Выход", "Закрыть приложение")

	// Запускаем основную логику
	startApp()

	// Обработка кликов
	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				openBrowser(fmt.Sprintf("http://localhost:%d", config.GlobalConfig.Generic.WebServerPort))
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

// openBrowser открывает URL в браузере по умолчанию
func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	}
	if err != nil {
		logging.Log.Errorf("Ошибка открытия браузера: %v", err)
	}
}
