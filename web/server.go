package web

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"venera/config"
	"venera/logging"
)

var server *http.Server

// StartWebServer запускает HTTP сервер
func StartWebServer() {
	port := config.GlobalConfig.Generic.WebServerPort
	mux := http.NewServeMux()
	
	SetupRoutes(mux)

	server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go func() {
		logging.Log.Infof("Веб-сервер запущен на порту %d", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logging.Log.Fatalf("Ошибка веб-сервера: %v", err)
		}
	}()
}

// StopWebServer останавливает HTTP сервер
func StopWebServer() {
	if server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			logging.Log.Errorf("Ошибка при остановке веб-сервера: %v", err)
		}
	}
}
