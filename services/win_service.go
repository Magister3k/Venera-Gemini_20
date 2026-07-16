package services

import (
	"fmt"
	"os"

	"github.com/kardianos/service"
	"golang.org/x/sys/windows"
	"venera/logging"
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

// getService возвращает объект службы kardianos/service (имя строго "VeneraSrv" по п.15.6 ТЗ)
func getService(startApp func(), stopApp func()) (service.Service, error) {
	svcConfig := &service.Config{
		Name:        "VeneraSrv",
		DisplayName: "Venera Collector",
		Description: "Система сбора идентификаторов в потоке пакетных данных (Tshark/Dragonfly/PostgreSQL).",
	}

	prg := &program{
		startFunc: startApp,
		stopFunc:  stopApp,
	}

	return service.New(prg, svcConfig)
}

// RunService настраивает и запускает приложение как службу Windows
func RunService(startApp func(), stopApp func()) error {
	s, err := getService(startApp, stopApp)
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

// InstallService устанавливает службу в ОС Windows (п.14, 15.6 ТЗ)
func InstallService() error {
	if !IsAdmin() {
		return fmt.Errorf("для установки службы требуются права Администратора (п.16.2 ТЗ)")
	}

	s, err := getService(nil, nil)
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

// UninstallService останавливает и удаляет службу из ОС Windows (п.14, 15.7 ТЗ)
func UninstallService() error {
	if !IsAdmin() {
		return fmt.Errorf("для удаления службы требуются права Администратора")
	}

	s, err := getService(nil, nil)
	if err != nil {
		return fmt.Errorf("ошибка инициализации конфигурации службы: %v", err)
	}

	status, _ := s.Status()
	if status == service.StatusUnknown {
		return fmt.Errorf("служба VeneraSrv не установлена")
	}

	// Попытка остановить перед удалением (п.15.7 ТЗ)
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

// ControlService позволяет запустить или остановить установленную службу (п.14 ТЗ)
func ControlService(action string) error {
	if !IsAdmin() {
		return fmt.Errorf("для управления службой требуются права Администратора")
	}

	s, err := getService(nil, nil)
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

// GetServiceStatus возвращает текущее состояние службы (п.9.1 ТЗ)
func GetServiceStatus() (string, error) {
	s, err := getService(nil, nil)
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

// IsAdmin проверяет, запущено ли приложение с правами Администратора (п.16 ТЗ)
func IsAdmin() bool {
	// Для Windows мы пытаемся открыть 물리ческий диск (PhysicalDrive) на чтение
	// или просто проверяем встроенную функцию из golang.org/x/sys/windows.
	// Ограничимся простой проверкой открытия токена доступа.
	var sid *windows.SID
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid,
	)
	if err != nil {
		return false
	}
	defer windows.FreeSid(sid)

	token := windows.Token(0)
	member, err := token.IsMember(sid)
	if err != nil {
		// Fallback: попытаемся открыть системный диск, доступный только админам
		_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
		return err == nil
	}
	return member
}
