package web

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"venera/config"
	"venera/data"
	"venera/models"
	"venera/processes"
	"venera/utils"
)

// SetupRoutes регистрирует все HTTP маршруты (REST API)
func SetupRoutes(app *fiber.App) {
	api := app.Group("/api")

	api.Get("/processes", apiGetProcesses)
	api.Post("/processes", apiAddProcess)
	api.Delete("/processes/:id", apiDeleteProcess)

	api.Post("/process/:id/action", apiProcessAction)

	api.Get("/config", apiGetConfig)
	api.Post("/config", apiUpdateConfig)

	// API для получения данных из итоговой БД (резервный REST интерфейс:
	// хотя ТЗ также требует WebSocket для этого, но REST удобнее для экспорта xlsx)
	api.Get("/db/export", apiExportDB)
}

// apiGetProcesses возвращает список всех процессов
func apiGetProcesses(c *fiber.Ctx) error {
	list := processes.GetAllProcesses()
	return c.JSON(list)
}

// apiAddProcess добавляет новый процесс в конфигурацию и сохраняет ее
func apiAddProcess(c *fiber.Ctx) error {
	var p models.ProcessConfig
	if err := c.BodyParser(&p); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "некорректный формат данных"})
	}

	if p.ID == "" {
		p.ID = utils.GenerateID()
	}

	err := processes.AddOrUpdateProcess(p)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(p)
}

// apiDeleteProcess удаляет процесс (и останавливает, если запущен)
func apiDeleteProcess(c *fiber.Ctx) error {
	id := c.Params("id")

	// Если запущен, сначала останавливаем
	if p, ok := processes.GetProcess(id); ok && p.Status == models.StatusRunning {
		_ = processes.Manager.StopProcess(id)
	}

	err := processes.RemoveProcess(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusOK)
}

// apiProcessAction запускает или останавливает процесс
func apiProcessAction(c *fiber.Ctx) error {
	id := c.Params("id")
	action := c.Query("action") // start или stop

	p, ok := processes.GetProcess(id)
	if !ok {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "процесс не найден"})
	}

	var err error
	switch action {
	case "start":
		err = processes.Manager.StartProcess(p)
	case "stop":
		err = processes.Manager.StopProcess(id)
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "неизвестное действие"})
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusOK)
}

// apiGetConfig возвращает конфигурацию системы
func apiGetConfig(c *fiber.Ctx) error {
	return c.JSON(config.GetConfig())
}

// apiUpdateConfig сохраняет новую конфигурацию
func apiUpdateConfig(c *fiber.Ctx) error {
	var cfg models.Config
	if err := c.BodyParser(&cfg); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "некорректный формат конфигурации"})
	}

	err := config.UpdateConfig(&cfg)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusOK)
}

// apiExportDB выгрузка данных из БД. В идеале генерирует xlsx,
// здесь мы возвращаем JSON-дамп для простоты интерфейса, фронтенд может сгенерировать excel.
func apiExportDB(c *fiber.Ctx) error {
	if data.PgPool == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "БД отключена"})
	}

	// Лимит на выгрузку (для безопасности)
	query := `SELECT source, key, value, date_first, date_last FROM venera_data LIMIT 1000;`
	rows, err := data.PgPool.Query(context.Background(), query)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var src, k, v string
		var df, dl interface{} // dates
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

	return c.JSON(results)
}
