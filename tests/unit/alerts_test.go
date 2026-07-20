package unit_test

import (
	"os"
	"testing"
	"venera/logging"
	"venera/notify"

	"github.com/sirupsen/logrus"
)

func init() {
	// Инициализация мок-логгера для тестов
	logging.Log = logrus.New()
	logging.Log.SetOutput(os.Stdout)
}

func TestCELAlertsLogic(t *testing.T) {
	// Создаем тестовый файл алертов generic.alr
	testFile := "test_generic.alr"
	content := `
// Формат: Message | Severity | CEL_Expression
Обнаружен админ | warning | value == 'admin' && key == 'username'
Критическая уязвимость | critical | source == 'net1' && value.contains('drop table')
Плохой IP | info | key == 'ip' && value.startsWith('192.168.100.')
`
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Ошибка создания тестового файла алертов: %v", err)
	}
	defer os.Remove(testFile)

	err = notify.LoadAlerts(testFile)
	if err != nil {
		t.Fatalf("LoadAlerts вернул ошибку: %v", err)
	}

	// Функция для проверки не возвращает значение, она логирует (ProcessAlert).
	// В реальном мире мы бы мокнули ProcessAlert или перехватывали вывод,
	// но мы просто вызываем и проверяем, что нет паники, и логика CEL компилируется правильно.

	notify.CheckAlerts("source1", "username", "admin")     // Должен сработать 1
	notify.CheckAlerts("net1", "sql", "drop table users;") // Должен сработать 2
	notify.CheckAlerts("net1", "ip", "192.168.100.5")      // Должен сработать 3
	notify.CheckAlerts("net1", "ip", "10.0.0.1")           // Не сработает

	// Простая проверка, что всё отработало без ошибок.
}
