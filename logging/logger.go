package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"venera/config"
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
	
	// Логирование версии (п.7.3)
	Log.Infof("Запуск Venera (версия будет здесь)")

	// Запуск ротации логов в фоне (п.7.1, 7.2)
	go startLogRotation(logDir, config.GlobalConfig.Generic.LogRotationDays)

	return nil
}
