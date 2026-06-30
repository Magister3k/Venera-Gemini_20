package web

import (
	"encoding/json"
	"net/http"
	"venera/config"
	"venera/processes"
)

// API: Получение списка процессов
func apiProcesses(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		list := processes.GetAllProcesses()
		json.NewEncoder(w).Encode(list)
	} else if r.Method == "POST" {
		// Парсинг нового процесса и добавление
		// ... (реализация парсинга)
	}
}

// API: Управление процессом
func apiProcessAction(w http.ResponseWriter, r *http.Request) {
	// action := r.URL.Query().Get("action")
	// id := r.URL.Query().Get("id")
	// switch action { "start": processes.Manager.StartProcess(...) }
}

// API: Настройки
func apiConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		json.NewEncoder(w).Encode(config.GlobalConfig)
	}
}

// Установка маршрутов
func SetupRoutes(mux *http.ServeMux) {
	// API
	mux.HandleFunc("/api/processes", apiProcesses)
	mux.HandleFunc("/api/process/action", apiProcessAction)
	mux.HandleFunc("/api/config", apiConfig)

	// Статика (SPA React)
	fs := http.FileServer(http.Dir("./react-ui/dst"))
	mux.Handle("/", fs)
}
