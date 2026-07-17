package logging

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// startLogRotation периодически проверяет старые логи и сжимает их в фоне (п.7.1, 7.2 ТЗ).
func startLogRotation(logDir string, keepDays int) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		files, err := os.ReadDir(logDir)
		if err != nil {
			if Log != nil {
				Log.Errorf("Ошибка чтения папки логов для ротации: %v", err)
			}
			continue
		}

		cutoff := time.Now().AddDate(0, 0, -keepDays)

		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".log") {
				continue
			}

			info, err := f.Info()
			if err != nil {
				continue
			}

			// Если файл старше cutoff (п.7.1)
			if info.ModTime().Before(cutoff) {
				logPath := filepath.Join(logDir, f.Name())
				// Сжатие логов в архив формата gz (п.7.2 ТЗ)
				gzPath := logPath + ".gz"

				err := compressFileGZ(logPath, gzPath)
				if err != nil {
					if Log != nil {
						Log.Errorf("Ошибка сжатия лога %s: %v", logPath, err)
					}
				} else {
					if Log != nil {
						Log.Infof("Старый лог %s успешно сжат в %s", f.Name(), gzPath)
					}
					_ = os.Remove(logPath) // Удаляем оригинал после успешного сжатия
				}
			}
		}
	}
}

// compressFileGZ сжимает файл используя стандартную библиотеку compress/gzip (п.7.2 ТЗ).
func compressFileGZ(src, dst string) error {
	inFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer inFile.Close()

	outFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// Используем только gzip, так как ТЗ требует "архив формата gz"
	// (TAR здесь избыточен для одиночного файла)
	gw := gzip.NewWriter(outFile)
	defer gw.Close()

	_, err = io.Copy(gw, inFile)
	return err
}
