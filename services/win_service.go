package services

import (
	"fmt"

	"golang.org/x/sys/windows/svc"
	"venera/logging"
)

type veneraService struct{}

func (m *veneraService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown

	changes <- svc.Status{State: svc.StartPending}
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	logging.Log.Infof("Служба VeneraSrv запущена")

	// Главный цикл службы
loop:
	for c := range r {
		switch c.Cmd {
		case svc.Interrogate:
			changes <- c.CurrentStatus
		case svc.Stop, svc.Shutdown:
			logging.Log.Infof("Получен сигнал остановки службы")
			break loop
		default:
			logging.Log.Warnf("Неожиданный сигнал для службы: %v", c)
		}
	}

	changes <- svc.Status{State: svc.StopPending}
	return
}

// RunService запускает приложение в режиме службы Windows
func RunService(startApp func(), stopApp func()) error {
	isInteractive, err := svc.IsAnInteractiveSession()
	if err != nil {
		return fmt.Errorf("ошибка определения сессии: %v", err)
	}

	if isInteractive {
		return fmt.Errorf("приложение не может быть запущено как служба в интерактивной сессии")
	}

	// Запускаем основную логику приложения
	go startApp()

	err = svc.Run("VeneraSrv", &veneraService{})
	
	// Останавливаем приложение при выходе
	stopApp()
	
	if err != nil {
		return fmt.Errorf("ошибка выполнения службы: %v", err)
	}

	return nil
}
