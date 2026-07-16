package web

import (
	"encoding/json"
	"fmt"
	"net/http"

	"venera/config"
	"venera/data"
	"venera/logging"
	"venera/processes"
)

var server *http.Server

// StartWebServer запускает HTTP сервер
func StartWebServer() {
	port := config.GlobalConfig.Generic.WebServerPort
	addr := fmt.Sprintf(":%d", port)

	mux := http.NewServeMux()
	
	// API routes
	mux.HandleFunc("/api/processes", handleProcesses)
	mux.HandleFunc("/api/data", handleData)
	
	// Статика
	fs := http.FileServer(http.Dir("web/static"))
	mux.Handle("/", fs)

	server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		logging.Log.Infof("Запуск веб-сервера на %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logging.Log.Fatalf("Ошибка веб-сервера: %v", err)
		}
	}()
}

// StopWebServer останавливает веб-сервер
func StopWebServer() {
	if server != nil {
		_ = server.Close()
	}
}

// handleProcesses обрабатывает статусы процессов
func handleProcesses(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	procs := processes.GetAllProcesses()
	json.NewEncoder(w).Encode(procs)
}

// handleData возвращает топ данных (витрина) из DragonflyDB
func handleData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	sourceID := r.URL.Query().Get("source")
	if sourceID == "" {
		http.Error(w, "source parametr required", http.StatusBadRequest)
		return
	}

	// Для упрощения показываем кол-во в очереди
	len, _ := data.GetListLength(sourceID)
	
	resp := map[string]interface{}{
		"source": sourceID,
		"queue":  len,
	}
	
	json.NewEncoder(w).Encode(resp)
}
