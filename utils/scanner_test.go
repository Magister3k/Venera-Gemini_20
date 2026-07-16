package utils_test

import (
	"os"
	"path/filepath"
	"testing"
	"venera/utils"
)

func TestGenerateID(t *testing.T) {
	id1 := utils.GenerateID()
	id2 := utils.GenerateID()

	if len(id1) != 16 {
		t.Errorf("Ожидаемая длина ID 16, получено %d", len(id1))
	}
	if id1 == id2 {
		t.Errorf("Сгенерированные ID не должны совпадать: %s == %s", id1, id2)
	}
}

func TestScanFiles(t *testing.T) {
	// Создаем временную структуру папок
	tempDir := t.TempDir()

	// Файлы в корне
	os.WriteFile(filepath.Join(tempDir, "test1.pcap"), []byte("data"), 0644)
	os.WriteFile(filepath.Join(tempDir, "test2.txt"), []byte("data"), 0644)

	// Подпапка
	subDir := filepath.Join(tempDir, "sub")
	os.Mkdir(subDir, 0755)
	os.WriteFile(filepath.Join(subDir, "test3.pcapng"), []byte("data"), 0644)

	// Тест 1: Сканирование без подпапок, без фильтрации
	files, err := utils.ScanFiles(tempDir, false, nil)
	if err != nil {
		t.Fatalf("Ошибка ScanFiles: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("Ожидалось 2 файла, найдено %d", len(files))
	}

	// Тест 2: Сканирование без подпапок, с фильтрацией
	files, err = utils.ScanFiles(tempDir, false, []string{".pcap"})
	if err != nil {
		t.Fatalf("Ошибка ScanFiles: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("Ожидался 1 файл, найдено %d", len(files))
	}

	// Тест 3: Сканирование с подпапками, с фильтрацией (pcap и pcapng)
	files, err = utils.ScanFiles(tempDir, true, []string{".pcap", ".pcapng"})
	if err != nil {
		t.Fatalf("Ошибка ScanFiles: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("Ожидалось 2 файла (test1.pcap, test3.pcapng), найдено %d", len(files))
	}
}
