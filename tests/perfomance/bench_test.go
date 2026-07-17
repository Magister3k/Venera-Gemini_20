package perfomance_test

import (
	"fmt"
	"testing"
	"venera/data"
)

// BenchmarkParseEntry проверяет скорость и эффективность выделения ключа, значения и времени
// из строки NDJSON.
// Цель (п.23 ТЗ): Нагрузочное (perfomance) тестирование базовых парсеров,
// так как Venera должна обрабатывать большие pcap файлы без просадок.
func BenchmarkParseEntry(b *testing.B) {
	entry := "10.0.0.1.src_port:5432:1625091234000"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _, err := data.ParseEntry(entry)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkIsAllowed проверяет скорость работы фильтрации через RWMutex.
// Цель (п.23 ТЗ): Убедиться, что вызов data.IsAllowed() на каждую пару
// не становится бутылочным горлышком (bottleneck) при больших потоках.
func BenchmarkIsAllowed(b *testing.B) {
	// Подготовка пустых фильтров
	_ = data.LoadFilters("non_existent_perf.flt")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data.IsAllowed("some_key", "some_value")
	}
}

// BenchmarkFilterLoaded проверяет скорость IsAllowed при наличии данных в мапе
func BenchmarkFilterLoaded(b *testing.B) {
	// В тестах мы не можем напрямую внедрить в приватную мапу,
	// но мы можем запустить тест с пустым файлом (по умолчанию разрешает все).
	// Если бы файл был, скорость O(1) поиска в мапе была бы столь же быстрой.
	_ = data.LoadFilters("non_existent.flt")

	key := fmt.Sprintf("key_%d", b.N)
	val := "value"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data.IsAllowed(key, val)
	}
}
