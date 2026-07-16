package services

import (
	"fmt"
	"venera/logging"

	"github.com/kardianos/service"
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

// RunService настраивает и запускает приложение как службу Windows
func RunService(startApp func(), stopApp func()) error {
	svcConfig := &service.Config{
		Name:        "VeneraService",
		DisplayName: "Venera Collector",
		Description: "Система сбора идентификаторов в потоке пакетных данных.",
	}

	prg := &program{
		startFunc: startApp,
		stopFunc:  stopApp,
	}

	s, err := service.New(prg, svcConfig)
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