package services

import (
	"fmt"

	"github.com/kardianos/service"

	"venera/logging"
	"venera/utils"
)

type program struct {
	startFunc func()
	stopFunc  func()
}

func (p *program) Start(s service.Service) error {
	logging.Log.Info("Запуск службы Venera...")
	go p.startFunc()
	return nil
}

func (p *program) Stop(s service.Service) error {
	logging.Log.Info("Остановка службы Venera...")
	p.stopFunc()
	return nil
}

// getSvc возвращает объект службы kardianos/service
func getSvc(startApp func(), stopApp func()) (service.Service, error) {
	svcConfig := &service.Config{
		Name:        "VeneraSrv",
		DisplayName: "Venera",
		Description: "Система сбора идентификаторов в потоке пакетных данных",
	}

	prg := &program{
		startFunc: startApp,
		stopFunc:  stopApp,
	}

	return service.New(prg, svcConfig)
}

// RunSvc настраивает и запускает приложение как службу Windows
func RunSvc(startApp func(), stopApp func()) error {
	s, err := getSvc(startApp, stopApp)
	if err != nil {
		return fmt.Errorf("ошибка создания службы: %v", err)
	}

	logger, err := s.Logger(nil)
	if err != nil {
		return fmt.Errorf("ошибка создания логгера службы: %v", err)
	}

	err = s.Run()
	if err != nil {
		logger.Error(err)
		return fmt.Errorf("ошибка работы службы: %v", err)
	}

	return nil
}

// InstallSvc устанавливает службу Windows
func InstallSvc() error {
	if !utils.IsAdmin() {
		return fmt.Errorf("для установки службы требуются права Администратора")
	}

	s, err := getSvc(nil, nil)
	if err != nil {
		return fmt.Errorf("ошибка инициализации конфигурации службы: %v", err)
	}

	status, _ := s.Status()
	if status != service.StatusUnknown {
		return fmt.Errorf("служба VeneraSrv уже установлена")
	}

	err = s.Install()
	if err != nil {
		return fmt.Errorf("ошибка установки службы: %v", err)
	}

	logging.Log.Info("Служба VeneraSrv успешно установлена.")
	return nil
}

// UninstallSvc останавливает и удаляет службу Windows
func UninstallSvc() error {
	if !utils.IsAdmin() {
		return fmt.Errorf("для удаления службы требуются права Администратора")
	}

	s, err := getSvc(nil, nil)
	if err != nil {
		return fmt.Errorf("ошибка инициализации конфигурации службы: %v", err)
	}

	status, _ := s.Status()
	if status == service.StatusUnknown {
		return fmt.Errorf("служба VeneraSrv не установлена")
	}

	// Попытка остановить перед удалением
	if status == service.StatusRunning {
		_ = s.Stop()
	}

	err = s.Uninstall()
	if err != nil {
		return fmt.Errorf("ошибка удаления службы: %v", err)
	}

	logging.Log.Info("Служба VeneraSrv успешно удалена.")
	return nil
}

// ControlSvc позволяет запустить или остановить установленную службу
func ControlSvc(action string) error {
	if !utils.IsAdmin() {
		return fmt.Errorf("для управления службой требуются права Администратора")
	}

	s, err := getSvc(nil, nil)
	if err != nil {
		return err
	}

	switch action {
	case "start":
		return s.Start()
	case "stop":
		return s.Stop()
	default:
		return fmt.Errorf("неизвестное действие: %s", action)
	}
}

// GetSvcStatus возвращает текущее состояние службы
func GetSvcStatus() (string, error) {
	s, err := getSvc(nil, nil)
	if err != nil {
		return "Ошибка", err
	}

	status, err := s.Status()
	if err != nil {
		return "Не установлена", nil
	}

	switch status {
	case service.StatusRunning:
		return "Запущена", nil
	case service.StatusStopped:
		return "Остановлена", nil
	default:
		return "Неизвестно", nil
	}
}