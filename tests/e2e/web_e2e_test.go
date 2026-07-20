package tests

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"venera/config"
	"venera/logging"
	"venera/web"
)

func init() {
	logging.Log = logrus.New()
	logging.Log.SetOutput(os.Stdout)
}

// TestE2EWebServer проверяет запуск и остановку веб-сервера (SPA Fiber)
// Цель: E2E проверка доступности главного интерфейса управления системой.
func TestE2EWebServer(t *testing.T) {
	t.Log("Запуск E2E теста веб-сервера")

	// Подготовка конфигурации
	config.GlobalConfig = config.DefaultConfig()
	config.GlobalConfig.Generic.WebServerPort = 8085 // Используем нестандартный порт

	// Запуск сервера
	go web.StartWebServer()

	// Ждем поднятия сервера
	time.Sleep(200 * time.Millisecond)

	// E2E запрос: Проверка доступности API процессов
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "http://127.0.0.1:8085/api/processes", nil)
	if err != nil {
		t.Fatalf("Ошибка создания HTTP запроса: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		t.Fatalf("Ошибка подключения к E2E Fiber серверу: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Ожидался статус 200 OK, получено: %v", resp.Status)
	}

	// Остановка сервера
	web.StopWebServer()
	t.Log("E2E тест успешно завершен.")
}
