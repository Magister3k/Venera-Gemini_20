package web

import (
	"fmt"
	"time"
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"

	"venera/metrics"
	"venera/logging"
	"venera/data"
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
			payload := metrics.CollectAllStats()
			if err := c.WriteJSON(payload); err != nil {
				return 
			}
		}
	}
}

// wsLogsHandler передает логи в реальном времени (п.10.8 ТЗ)
func wsLogsHandler(c *websocket.Conn) {
	defer c.Close()
	
	// Подписываемся на броадкаст логов
	ch := logging.Hub.Subscribe()
	defer logging.Hub.Unsubscribe(ch)

	// Запускаем горутину для чтения из сокета, чтобы реагировать на закрытие соединения клиентом
	go func() {
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				c.Close()
				break
			}
		}
	}()

	for {
		msg, ok := <-ch
		if !ok {
			break
		}
		if err := c.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
			break
		}
	}
}

// wsDBHandler передает данные из БД с фильтрацией (п.10.6 ТЗ)
func wsDBHandler(c *websocket.Conn) {
	defer c.Close()
	
	type FilterRequest struct {
		Search string `json:"search"`
		Limit  int    `json:"limit"`
	}

	for {
		var req FilterRequest
		if err := c.ReadJSON(&req); err != nil {
			break
		}

		if data.PgPool == nil {
			c.WriteJSON(map[string]string{"error": "БД отключена"})
			continue
		}

		limit := req.Limit
		if limit <= 0 || limit > 500 {
			limit = 50
		}

		query := `SELECT source, key, value, date_first, date_last FROM venera_data `
		args := []interface{}{}
		
		if req.Search != "" {
			query += `WHERE source ILIKE $1 OR key ILIKE $1 OR value ILIKE $1 `
			args = append(args, "%"+req.Search+"%")
		}
		
		query += fmt.Sprintf(`LIMIT %d`, limit)

		rows, err := data.PgPool.Query(context.Background(), query, args...)
		if err != nil {
			c.WriteJSON(map[string]string{"error": err.Error()})
			continue
		}

		var results []map[string]interface{}
		for rows.Next() {
			var src, k, v string
			var df, dl interface{}
			if err := rows.Scan(&src, &k, &v, &df, &dl); err == nil {
				results = append(results, map[string]interface{}{
					"source":     src,
					"key":        k,
					"value":      v,
					"date_first": df,
					"date_last":  dl,
				})
			}
		}
		rows.Close()

		c.WriteJSON(results)
	}
}
