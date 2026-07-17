package web

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"venera/config"
	"venera/logging"
)

var app *fiber.App

// StartWebServer запускает веб-сервер на базе фреймворка fiber (п.10.2 ТЗ).
func StartWebServer() {
	cfg := config.GetConfig()
	port := cfg.Generic.WebServerPort
	addr := fmt.Sprintf(":%d", port)

	// Инициализация Fiber App
	app = fiber.New(fiber.Config{
		AppName:      "Venera Web UI",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	})

	// Middleware
	app.Use(cors.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))

	// Установка API маршрутов
	SetupRoutes(app)
	// Установка WebSocket маршрутов (п.19.1 ТЗ)
	SetupWebSockets(app)

	// Раздача статических файлов SPA React (п.10.2 ТЗ)
	// SPA с поддержкой хэш-роутинга: отдаем index.html для всех неизвестных путей,
	// но fiber Static сам обслуживает index.html.
	app.Static("/", "./react-ui/dst", fiber.Static{
		Compress:  true,
		ByteRange: true,
		Browse:    false,
		Index:     "index.html",
	})

	go func() {
		logging.Log.Infof("Запуск веб-сервера Fiber на %s", addr)
		if err := app.Listen(addr); err != nil {
			logging.Log.Fatalf("Ошибка веб-сервера: %v", err)
		}
	}()
}

// StopWebServer корректно останавливает веб-сервер
func StopWebServer() {
	if app != nil {
		logging.Log.Info("Остановка веб-сервера Fiber...")
		_ = app.Shutdown()
	}
}
