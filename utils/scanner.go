package utils

import (
	"os"
	"path/filepath"
	"strings"
)

// ScanFiles возвращает список файлов в указанной директории.
// Реализует поддержку режима сканирования подпапок согласно п.13 ТЗ.
// Опционально фильтрует файлы по расширению (например, []string{".pcap", ".pcapng"}).
// Если allowedExts пуст, возвращает все файлы.
func ScanFiles(root string, scanSubfolders bool, allowedExts []string) ([]string, error) {
	var files []string

	// Вспомогательная функция проверки расширения
	isValidExt := func(filename string) bool {
		if len(allowedExts) == 0 {
			return true
		}
		ext := strings.ToLower(filepath.Ext(filename))
		for _, allowed := range allowedExts {
			if ext == strings.ToLower(allowed) {
				return true
			}
		}
		return false
	}

	// Обычное сканирование (без подпапок)
	if !scanSubfolders {
		entries, err := os.ReadDir(root)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if !e.IsDir() && isValidExt(e.Name()) {
				files = append(files, filepath.Join(root, e.Name()))
			}
		}
		return files, nil
	}

	// Рекурсивное сканирование подпапок
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Пропускаем папки, к которым нет доступа (во избежание прерывания сканирования)
			if os.IsPermission(err) {
				return nil
			}
			return err
		}
		if !info.IsDir() && isValidExt(info.Name()) {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}

