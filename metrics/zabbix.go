package metrics

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ZabbixExportHandler предоставляет метрики в формате JSON для активного/пассивного сбора Zabbix.
func ZabbixExportHandler(w http.ResponseWriter, r *http.Request) {
	// Собираем все текущие метрики
	stats := CollectAllStats()

	w.Header().Set("Content-Type", "application/json")

	// Конвертируем структуру StatsPayload в JSON
	jsonData, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка генерации метрик Zabbix: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

// Описание шаблона конфигурации Zabbix UserParameter или HTTP Agent можно найти в файле README
