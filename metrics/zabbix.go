package metrics

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ZabbixExportHandler предоставляет метрики (из п.6) в формате JSON для активного/пассивного сбора Zabbix.
// ТЗ п.18.6: "Вывод метрик (из п.6) для программы Zabbix."
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

// Шаблон конфигурации Zabbix UserParameter или HTTP Agent можно найти в README проекта
// как того требует п.24 ТЗ: "Добавить шаблон для использования метрик в программе Zabbix".
// (Файл zabbix_template.xml может быть сгенерирован или приложен в документации).
