package config_test

import (
	"os"
	"testing"

	"venera/config"
)

func TestConfigLoadAndSave(t *testing.T) {
	// Сохраняем оригинальный путь
	originalPath := config.ConfigPath
	tempFile := "config_test_temp.toml"
	config.ConfigPath = tempFile
	defer func() {
		config.ConfigPath = originalPath
		_ = os.Remove(tempFile)
		_ = os.Remove(tempFile + ".bak")
	}()

	// 1. Тест дефолтной конфигурации
	cfg := config.DefaultConfig()
	if cfg.Generic.MaxProcesses != 20 {
		t.Errorf("ожидалось MaxProcesses = 20, получено %d", cfg.Generic.MaxProcesses)
	}

	// 2. Тест сохранения
	err := config.SaveConfig(cfg)
	if err != nil {
		t.Fatalf("ошибка сохранения конфигурации: %v", err)
	}

	// Проверяем существование файла
	if _, err := os.Stat(tempFile); os.IsNotExist(err) {
		t.Fatalf("файл конфигурации не был создан")
	}

	// 3. Тест загрузки
	err = config.LoadConfig()
	if err != nil {
		t.Fatalf("ошибка загрузки конфигурации: %v", err)
	}

	loaded := config.GetConfig()
	if loaded.Generic.WebServerPort != 8080 {
		t.Errorf("ожидался порт веб-сервера 8080, получен %d", loaded.Generic.WebServerPort)
	}

	// 4. Тест валидации (неверный режим)
	loaded.Generic.Mode = "invalid_mode"
	err = config.UpdateConfig(&loaded)
	if err == nil {
		t.Error("ожидалась ошибка валидации для неверного режима работы")
	}

	// 5. Тест валидации (превышение лимита процессов)
	loaded.Generic.Mode = "tray"
	loaded.Generic.MaxProcesses = 21
	err = config.UpdateConfig(&loaded)
	if err == nil {
		t.Error("ожидалась ошибка валидации при MaxProcesses > 20")
	}

	// 6. Тест успешного обновления
	loaded.Generic.MaxProcesses = 15
	err = config.UpdateConfig(&loaded)
	if err != nil {
		t.Fatalf("ошибка корректного обновления конфигурации: %v", err)
	}

	finalCfg := config.GetConfig()
	if finalCfg.Generic.MaxProcesses != 15 {
		t.Errorf("ожидалось MaxProcesses = 15, получено %d", finalCfg.Generic.MaxProcesses)
	}

	// Проверяем наличие бэкапа
	if _, err := os.Stat(tempFile + ".bak"); os.IsNotExist(err) {
		t.Error("резервная копия конфигурации (.bak) не была создана")
	}
}
