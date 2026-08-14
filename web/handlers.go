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
	api.Get("/processes", apiGetAllProcs)
	api.Post("/processes", apiAddProc)
	api.Delete("/processes/:id", apiDelProc)
	api.Post("/process/:id/action", apiProcAction)
	api.Post("/processes/actions", apiAllProcsAction)

	// --- Настройки ---
	api.Get("/config", apiGetCfg)
	api.Post("/config", apiUpdCfg)
	api.Post("/config/test-postgres", apiTestPgDbConn)
	api.Post("/config/test-cachedb", apiTestCacheDbConn)

	// --- База Данных ---
	api.Get("/db/export", apiExportPgDb)
	api.Get("/db/export/xlsx", apiExportPgDbXlsx)

	// --- Диагностика ---
	api.Post("/diagnose/run", apiRunDiag)
	api.Get("/diagnose/pdf", apiDownloadDiagPDF)
	api.Get("/diagnose/archive", apiDownloadDiagArch)

	// --- Справка ---
	api.Get("/help", apiGetHelp)
	api.Get("/about", apiGetAbout)

	// Эндпоинт для Zabbix
	api.Get("/metrics", apiGetMetrics)
}

// apiGetAllProcs возвращает список всех процессов
func apiGetAllProcs(c *fiber.Ctx) error {
	list := processes.GetAllProcs()
	return c.JSON(list)
}

// apiAddProc добавляет новый процесс в конфигурацию и сохраняет ее
func apiAddProc(c *fiber.Ctx) error {
	var p models.ProcCfg
	if err := c.BodyParser(&p); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "некорректный формат данных"})
	}

	if p.ID == "" {
		p.ID = utils.GenerateID()
	}

	err := processes.AddOrUpdateProc(p)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(p)
}

// apiDelProc удаляет процесс (и останавливает, если запущен)
func apiDelProc(c *fiber.Ctx) error {
	id := c.Params("id")

	// Если запущен, сначала останавливаем
	if p, ok := processes.GetProc(id); ok && p.Status == models.StatusRunning {
		_ = processes.Manager.StopProc(id)
	}

	err := processes.RemoveProc(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusOK)
}

// apiProcAction запускает или останавливает процесс
func apiProcAction(c *fiber.Ctx) error {
	id := c.Params("id")
	action := c.Query("action") // start или stop

	p, ok := processes.GetProc(id)
	if !ok {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "процесс не найден"})
	}

	var err error
	switch action {
	case "start":
		err = processes.Manager.StartProc(p)
	case "stop":
		err = processes.Manager.StopProc(id)
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "неизвестное действие"})
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusOK)
}

// apiAllProcsAction запускает, останавливает или удаляет ВСЕ процессы
func apiAllProcsAction(c *fiber.Ctx) error {
	action := c.Query("action")
	procs := processes.GetAllProcs()

	for _, p := range procs {
		switch action {
		case "start":
			if p.Status != models.StatusRunning {
				_ = processes.Manager.StartProc(p)
			}
		case "stop":
			if p.Status == models.StatusRunning {
				_ = processes.Manager.StopProc(p.ID)
			}
		case "delete":
			if p.Status == models.StatusRunning {
				_ = processes.Manager.StopProc(p.ID)
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

// apiUpdCfg сохраняет новую конфигурацию
func apiUpdCfg(c *fiber.Ctx) error {
	var cfg models.Config
	if err := c.BodyParser(&cfg); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "некорректный формат конфигурации"})
	}

	err := config.UpdCfg(&cfg)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusOK)
}

// apiTestPgConn временно проверяет коннект к БД по переданным параметрам
func apiTestPgConn(c *fiber.Ctx) error {
	var cfg models.PgDbCfg
	if err := c.BodyParser(&cfg); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "некорректный формат"})
	}
	err := data.PingPgDb(&cfg)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

// apiTestCacheDbConn временно проверяет коннект к кэшу
func apiTestCacheDbConn(c *fiber.Ctx) error {
	var cfg models.CacheDbCfg
	if err := c.BodyParser(&cfg); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "некорректный формат"})
	}
	err := data.PingCacheDb(&cfg)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"status": "ok"})
}


// apiExportPgDb выгрузка данных из итоговой БД в JSON.
func apiExportPgDb(c *fiber.Ctx) error {
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

// apiExportPgDbXlsx потоковая выгрузка данных из итоговой БД в формате Excel
func apiExportPgDbXlsx(c *fiber.Ctx) error {
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

func apiRunDiag(c *fiber.Ctx) error {
	report, err := diagnose.RunDiag()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(report)
}

func apiDownloadDiagPDF(c *fiber.Ctx) error {
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

func apiDownloadDiagArch(c *fiber.Ctx) error {
	report, _ := diagnose.RunDiag() // Игнорируем ошибку, так как pdf мы все равно соберем
	
	pdfPath := "Venera_Diagnostic_Report.pdf"
	_ = diagnose.ExportReportPDF(report, pdfPath)
	defer os.Remove(pdfPath)

	gzPath := "Venera_Diagnostic_Archive.tar.gz"
	err := diagnose.CreateArchGZ(pdfPath, gzPath)
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
		"app_name": "Venera",
		"version": "1.0.0",
		"developer": "PineCode Lab",
		"build_time": "2026-07-21",
		"license": "Проприетарная (Для внутреннего использования)",
		"copyright": "© 2026 PineCode Lab. Все права защищены.",
	})
}

// apiGetMetrics отдает метрики системы и процессов в формате JSON для Zabbix
func apiGetMetrics(c *fiber.Ctx) error {
	stats := metrics.CollectAllStats()
	return c.JSON(stats)
}
