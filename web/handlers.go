package web

import (
	"context"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/xuri/excelize/v2"
	"venera/config"
	"venera/data"
	"venera/diagnose"
	"venera/metrics"
	"venera/models"
	"venera/processes"
	"venera/utils"
)

// SetupRoutes регистрирует все HTTP маршруты (REST API)
func SetupRoutes(app *fiber.App) {
	api := app.Group("/api")

	// --- Процессы ---
	api.Get("/processes", apiGetProcesses)
	api.Post("/processes", apiAddProcess)
	api.Delete("/processes/:id", apiDeleteProcess)
	api.Post("/process/:id/action", apiProcessAction)
	api.Post("/processes/actions", apiGroupProcessAction)

	// --- Настройки ---
	api.Get("/config", apiGetCfg)
	api.Post("/config", apiUpdateConfig)
	api.Post("/config/test-postgres", apiTestPostgres)
	api.Post("/config/test-dragonfly", apiTestDragonfly)

	// --- База Данных ---
	api.Get("/db/export", apiExportDB)
	api.Get("/db/export/xlsx", apiExportDBXlsx)

	// --- Диагностика ---
	api.Post("/diagnose/run", apiRunDiagnose)
	api.Get("/diagnose/pdf", apiDownloadDiagnosePDF)
	api.Get("/diagnose/archive", apiDownloadDiagnoseArchive)

	// --- Справка ---
	api.Get("/help", apiGetHelp)
	api.Get("/about", apiGetAbout)

	// Эндпоинт для Zabbix (п.18.6 ТЗ)
	api.Get("/metrics", apiGetMetrics)
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

// apiGroupProcessAction запускает, останавливает или удаляет ВСЕ процессы
func apiGroupProcessAction(c *fiber.Ctx) error {
	action := c.Query("action")
	procs := processes.GetAllProcesses()

	for _, p := range procs {
		switch action {
		case "start":
			if p.Status != models.StatusRunning {
				_ = processes.Manager.StartProcess(p)
			}
		case "stop":
			if p.Status == models.StatusRunning {
				_ = processes.Manager.StopProcess(p.ID)
			}
		case "delete":
			if p.Status == models.StatusRunning {
				_ = processes.Manager.StopProcess(p.ID)
			}
			_ = processes.RemoveProcess(p.ID)
		}
	}
	return c.SendStatus(fiber.StatusOK)
}

// apiGetCfg возвращает конфигурацию системы
func apiGetCfg(c *fiber.Ctx) error {
	return c.JSON(config.GetCfg())
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

// apiTestPostgres временно проверяет коннект к БД по переданным параметрам
func apiTestPostgres(c *fiber.Ctx) error {
	var cfg models.PostgreSQLConfig
	if err := c.BodyParser(&cfg); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "некорректный формат"})
	}
	err := data.PingPostgres(&cfg)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

// apiTestDragonfly временно проверяет коннект к кэшу
func apiTestDragonfly(c *fiber.Ctx) error {
	var cfg models.DragonflyDBConfig
	if err := c.BodyParser(&cfg); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "некорректный формат"})
	}
	err := data.PingDragonfly(&cfg)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"status": "ok"})
}


// apiExportDB выгрузка данных из БД в JSON.
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

	return c.JSON(results)
}

// apiExportDBXlsx потоковая выгрузка данных из БД в формате Excel
func apiExportDBXlsx(c *fiber.Ctx) error {
	if data.PgPool == nil {
		return c.Status(fiber.StatusServiceUnavailable).SendString("БД отключена")
	}

	f := excelize.NewFile()
	defer f.Close()

	sw, err := f.NewStreamWriter("Sheet1")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	// Заголовки
	err = sw.SetRow("A1", []interface{}{"Источник", "Ключ", "Значение", "Первое появление", "Последнее появление"})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	query := `SELECT source, key, value, date_first, date_last FROM venera_data`
	rows, err := data.PgPool.Query(context.Background(), query)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}
	defer rows.Close()

	rowID := 2
	for rows.Next() {
		var src, k, v string
		var df, dl interface{}
		if err := rows.Scan(&src, &k, &v, &df, &dl); err == nil {
			cell, _ := excelize.CoordinatesToCellName(1, rowID)
			_ = sw.SetRow(cell, []interface{}{src, k, v, df, dl})
			rowID++
		}
	}
	if err := sw.Flush(); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	c.Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Set("Content-Disposition", "attachment; filename=venera_export.xlsx")
	
	buf, _ := f.WriteToBuffer()
	return c.SendStream(buf)
}

func apiRunDiagnose(c *fiber.Ctx) error {
	report, err := diagnose.RunDiag()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(report)
}

func apiDownloadDiagnosePDF(c *fiber.Ctx) error {
	report, err := diagnose.RunDiag()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}
	
	pdfPath := "Venera_Diagnostic_Report.pdf"
	err = diagnose.ExportReportPDF(report, pdfPath)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}
	
	defer os.Remove(pdfPath)
	return c.Download(pdfPath)
}

func apiDownloadDiagnoseArchive(c *fiber.Ctx) error {
	report, _ := diagnose.RunDiag() // Игнорируем ошибку, так как pdf мы все равно соберем
	
	pdfPath := "Venera_Diagnostic_Report.pdf"
	_ = diagnose.ExportReportPDF(report, pdfPath)
	defer os.Remove(pdfPath)

	gzPath := "Venera_Diagnostic_Archive.tar.gz"
	err := diagnose.CreateArchiveGZ(pdfPath, gzPath)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}
	
	// Fiber Download сам удалит файл после скачивания если передать нужные функции,
	// но мы просто удалим его через defer, после того как SendFile прочитает его
	// (в Fiber Download работает асинхронно, лучше отдавать буфером или не удалять пока не отдали).
	return c.Download(gzPath)
}

func apiGetHelp(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"sections": []map[string]string{
			{"title": "Введение", "content": "Venera — это высокопроизводительная система сбора и фильтрации сетевых идентификаторов в реальном времени."},
			{"title": "Управление процессами", "content": "Для запуска сбора создайте процесс в панели 'Процессы'. Укажите тип источника (сетевой интерфейс, папка или файл PCAP) и нажмите 'Старт'."},
			{"title": "Фильтрация и контроль", "content": "Фильтрация данных производится по спискам generic.flt (белый список ключей и черный список значений)."},
		},
	})
}

func apiGetAbout(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"app_name": "Venera Collector",
		"version": "1.0.0",
		"developer": "Venera Team",
		"build_time": "2026-07-21",
		"license": "Проприетарная (Для внутреннего использования)",
		"copyright": "© 2026 Venera Team. Все права защищены.",
	})
}

// apiGetMetrics отдает метрики системы и процессов в формате JSON для Zabbix (п.18.6 ТЗ)
func apiGetMetrics(c *fiber.Ctx) error {
	stats := metrics.CollectAllStats()
	return c.JSON(stats)
}
