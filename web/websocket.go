package web

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"

	"venera/metrics"
)

// SetupWebSockets регистрирует маршруты для WebSocket (п.19.1 ТЗ)
func SetupWebSockets(app *fiber.App) {
	// Middleware для обновления протокола до WebSocket
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	// Эндпоинт статистики реального времени (п.10.5 ТЗ)
	app.Get("/ws/statistics", websocket.New(wsStatisticsHandler))

	// Эндпоинт для логов (п.10.8 ТЗ)
	app.Get("/ws/logs", websocket.New(wsLogsHandler))

	// Эндпоинт для базы данных (п.10.6 ТЗ)
	app.Get("/ws/db", websocket.New(wsDBHandler))
}

// wsStatisticsHandler передает метрики системы и процессов каждую секунду (п.1.13, 10.5 ТЗ)
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

// wsLogsHandler передает логи в реальном времени (п.10.8 ТЗ)
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

// wsDBHandler передает данные из БД или статус (п.10.6 ТЗ)
func wsDBHandler(c *websocket.Conn) {
	defer c.Close()
	// ТЗ п.10.6: "получение и вывод данных из базы СУБД PostgreSQL" через веб-сокет.
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
