package metrics

import (
	"context"
	"fmt"
	"venera/data"
)

// GetDbSizes собирает размеры баз данных средствами самих СУБД.
func GetDbSizes() (uint64, uint64, error) {
	var cacheDbSize uint64
	var pgDbSize    uint64

	// Размер кэширующей СУБД
	if data.CacheDbClient != nil {
		// Для Redis/DragonflyDB используем команду MEMORY USAGE или INFO memory.
		// В DragonflyDB команда INFO memory выдает used_memory в байтах.
		infoStr, err := data.CacheDbClient.Info(context.Background(), "memory").Result()
		if err == nil {
			cacheDbSize = parseInfoMemory(infoStr, "used_memory:")
		}
	}

	// Размер базы в СУБД PostgreSQL
	if data.PgPool != nil {		
		// Используем встроенную функцию pg_database_size
		query := `SELECT pg_database_size(current_database());`
		_ = data.PgPool.QueryRow(context.Background(), query).Scan(&pgDbSize)
	}

	return cacheDbSize, pgDbSize, nil
}

// parseInfoMemory парсит вывод Info{"memory") Redis/Dragonfly
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
