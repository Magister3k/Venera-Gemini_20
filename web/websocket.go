package web

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"

	"venera/metrics"
)

// SetupWebSockets регистрирует маршруты для WebSocket
func SetupWebSockets(app *fiber.App) {
	// Middleware для обновления протокола до WebSocket
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	// Эндпоинт статистики реального времени
	app.Get("/ws/statistics", websocket.New(wsStatisticsHandler))

	// Эндпоинт для логов
	app.Get("/ws/logs", websocket.New(wsLogsHandler))

	// Эндпоинт для базы данных
	app.Get("/ws/db", websocket.New(wsDBHandler))
}

// wsStatisticsHandler передает метрики системы и процессов каждую секунду
func wsStatisticsHandler(c *websocket.Conn) {
	defer c.Close()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Сбор статистики из модуля metrics
			payload := metrics.CollectAllStats()

			// Отправка JSON клиенту
			if err := c.WriteJSON(payload); err != nil {
				log.Printf("Отключение клиента WebSocket (statistics): %v", err)
				return // Выход при разрыве соединения
			}
		}
	}
}

// wsLogsHandler передает логи в реальном времени
func wsLogsHandler(c *websocket.Conn) {
	defer c.Close()
	// TODO: Реализовать подписку на систему логирования и передачу строк.
	// Для этого потребуется добавить Hook в logrus, который будет рассылать
	// сообщения в каналы подписанных WebSocket клиентов.
	// Оставляем базовую поддержку соединения.
	for {
		if _, _, err := c.ReadMessage(); err != nil {
			break
		}
	}
}

// wsDBHandler передает данные из БД или статус
func wsDBHandler(c *websocket.Conn) {
	defer c.Close()
	// Получение и вывод данных из базы СУБД PostgreSQL через веб-сокет.
	// Можно реализовать прием запросов на фильтрацию от клиента и отправку результатов.

	for {
		// Чтение фильтров/запросов от клиента
		_, _, err := c.ReadMessage()
		if err != nil {
			break
		}

		// Отправка ответа (заглушка: в реальности здесь будет выполнение SQL-запроса)
		// ...
	}
}
