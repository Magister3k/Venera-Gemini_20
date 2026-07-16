package manifest_test

import (
	"os"
	"testing"

	"venera/manifest"
)

func TestManifestConstants(t *testing.T) {
	if manifest.ManifestFilePath != "manifest.xml" {
		t.Errorf("Ожидался путь manifest.xml")
	}
	if manifest.EventIDManifestUpdated != 2003 {
		t.Errorf("Ожидался EventID 2003")
	}
}

func TestBackupManifest(t *testing.T) {
	// Создаем временный файл
	testManifest := "manifest.xml"
	manifest.ManifestFilePath = testManifest
	
	err := os.WriteFile(testManifest, []byte("<xml/>"), 0644)
	if err != nil {
		t.Fatalf("Ошибка создания тестового манифеста")
	}
	
	defer func() {
		os.Remove(testManifest)
		os.Remove(manifest.ManifestBackupPath)
		manifest.ManifestFilePath = "manifest.xml" // Восстанавливаем
	}()

	// Вызов приватной функции недоступен, но мы можем проверить Rollback, который требует бэкап
	// Сымитируем наличие бэкапа
	err = os.WriteFile(manifest.ManifestBackupPath, []byte("<xml>backup</xml>"), 0644)
	if err != nil {
		t.Fatalf("Ошибка создания тестового бэкапа")
	}

	// Попытаемся откатить (это вызовет ошибку wevtutil, поэтому перехватываем аккуратно)
	err = manifest.RollbackManifest()
	if err != nil {
		// Мы ожидаем ошибку wevtutil, так как он не зарегистрирован в системе
		t.Logf("Ожидаемая ошибка при вызове wevtutil в Rollback: %v", err)
	}

	// Проверим, что файл восстановился
	data, _ := os.ReadFile(testManifest)
	if string(data) != "<xml>backup</xml>" {
		t.Errorf("Rollback не восстановил файл корректно")
	}
}
