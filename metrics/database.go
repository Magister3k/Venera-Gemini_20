package metrics

import (
	"context"
	"fmt"
	"venera/data"
)

// GetDatabaseSizes собирает размеры баз данных средствами самих СУБД.
func GetDatabaseSizes() (uint64, uint64, error) {
	var cachedbSize uint64
	var pgdbSize uint64

	// Размер кэширующей СУБД
	if data.DragonflyClient != nil {
		// Для Redis/DragonflyDB используем команду MEMORY USAGE или INFO memory.
		// В DragonflyDB команда INFO memory выдает used_memory в байтах.
		infoStr, err := data.DragonflyClient.Info(context.Background(), "memory").Result()
		if err == nil {
			cachedbSize = parseInfoMemory(infoStr, "used_memory:")
		}
	}

	// Размер базы в СУБД PostgreSQL
	if data.PgPool != nil {		
		// Используем встроенную функцию pg_database_size
		query := `SELECT pg_database_size(current_database());`
		err := data.PgPool.QueryRow(context.Background(), query).Scan(&pgdbSize)
		//if err != nil {
			// Если запрос упал, просто вернем 0 или можно залогировать (логирование в collector.go)
		//}
	}

	return cachedbSize, pgdbSize, err
}

// parseInfoMemory парсит вывод INFO memory Redis/Dragonfly
func parseInfoMemory(info, key string) uint64 {
	// info это текст с переносами строк
	// ищем строку с key (например, "used_memory:123456")
	lines := splitLines(info)
	for _, line := range lines {
		if startsWith(line, key) {
			valStr := line[len(key):]
			// убираем возможные пробелы/возвраты каретки
			valStr = trimSpace(valStr)
			var val uint64
			fmt.Sscanf(valStr, "%d", &val)
			return val
		}
	}
	return 0
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[0:len(prefix)] == prefix
}

func trimSpace(s string) string {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\r' || s[i] == '\n') {
		i++
	}
	j := len(s)
	for j > i && (s[j-1] == ' ' || s[j-1] == '\r' || s[j-1] == '\n') {
		j--
	}
	return s[i:j]
}
