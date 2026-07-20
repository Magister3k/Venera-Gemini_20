package unit_test

import (
	"github.com/sirupsen/logrus"
	"os"
	"testing"
	"venera/data"
	"venera/logging"
)

func init() {
	// Инициализация мок-логгера для тестов
	logging.Log = logrus.New()
	logging.Log.SetOutput(os.Stdout)
}

func TestFilterLogic(t *testing.T) {
	// Создаем тестовый файл фильтров
	testFile := "test_generic.flt"
	content := `+ key1|Название 1
- value1
- value2
+ key2|Название 2
+ key3|Название 3
- bad_value
`
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Ошибка создания тестового файла фильтров: %v", err)
	}
	defer os.Remove(testFile)

	err = data.LoadFilters(testFile)
	if err != nil {
		t.Fatalf("LoadFilters вернул ошибку: %v", err)
	}

	keys := data.GetFilterKeys()
	if len(keys) != 3 {
		t.Errorf("Ожидалось 3 ключа в белом списке, получено: %d", len(keys))
	}

	// Тест 1: Ключ отсутствует в белом списке
	if data.IsAllowed("unknown_key", "any_value") {
		t.Error("Ключ unknown_key должен быть отброшен (нет в белом списке)")
	}

	// Тест 2: Ключ в белом списке, значение в черном списке
	if data.IsAllowed("key1", "value1") {
		t.Error("Значение value1 для key1 должно быть отброшено (в черном списке)")
	}

	// Тест 3: Ключ в белом списке, значение разрешено (нет в черном)
	if !data.IsAllowed("key1", "good_value") {
		t.Error("Значение good_value для key1 должно быть разрешено")
	}

	// Тест 4: Ключ в белом списке, черного списка для него нет
	if !data.IsAllowed("key2", "any_value") {
		t.Error("Значение any_value для key2 должно быть разрешено (черный список пуст)")
	}
}

func TestControlListLogic(t *testing.T) {
	// Создаем тестовый файл контроля
	testFile := "test_generic.ctr"
	content := `key1|value1
key2|value2
`
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Ошибка создания тестового файла контроля: %v", err)
	}
	defer os.Remove(testFile)

	err = data.LoadControlList(testFile)
	if err != nil {
		t.Fatalf("LoadControlList вернул ошибку: %v", err)
	}

	// Проверки
	if !data.IsOnControl("key1", "value1") {
		t.Error("Пара key1|value1 должна быть на контроле")
	}
	if data.IsOnControl("key1", "value2") {
		t.Error("Пара key1|value2 НЕ должна быть на контроле")
	}
	if data.IsOnControl("unknown", "value1") {
		t.Error("Пара unknown|value1 НЕ должна быть на контроле")
	}
}
