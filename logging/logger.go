package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"venera/config"
	"venera/manifest" // для получения текушей версии приложения
)

var Log *logrus.Logger

// InitLogger инициализирует централизованное логирование (logrus).
func InitLogger() error {
	Log = logrus.New()
	Log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	logDir := "Logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("ошибка создания папки логов: %v", err)
	}

	// Имя файла: год-месяц-число_время_номер.log
	now := time.Now()
	fileName := fmt.Sprintf("%s_%02d.log", now.Format("2006-01-02_15-04"), 1)
	logPath := filepath.Join(logDir, fileName)

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("ошибка открытия файла лога: %v", err)
	}

	// Пишем в файл и в консоль
	mw := io.MultiWriter(os.Stdout, file)
	Log.SetOutput(mw)

	// Получаем потокобезопасную конфигурацию
	cfg := config.GetConfig()

	// Интеграция с Windows Event Log
	eventHook, err := NewEventLogHook("VeneraApp")
	if err == nil {
		Log.AddHook(eventHook)
	} else {
		// Права администратора могут отсутствовать для регистрации источника в реестре
		Log.Warnf("Не удалось подключить хук Windows Event Log (требуются права администратора?): %v", err)
	}

	// Логирование версии
	Log.Infof("Запуск Venera (Версия: %s)", manifest.CurrentAppVersion)

	Log.AddHook(&WsLogHook{})

	// Запуск ротации логов в фоне
	// Используем дни из конфигурации. Если 0 - ставим дефолтные 7 дней
	rotationDays := cfg.Generic.LogRotationDays
	if rotationDays <= 0 {
		rotationDays = 7
	}
	go startLogRotation(logDir, rotationDays)

	return nil
}
