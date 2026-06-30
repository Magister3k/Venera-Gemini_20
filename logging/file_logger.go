package logging

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// startLogRotation периодически проверяет старые логи и сжимает их.
func startLogRotation(logDir string, keepDays int) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		files, err := os.ReadDir(logDir)
		if err != nil {
			Log.Errorf("Ошибка чтения папки логов для ротации: %v", err)
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

			if info.ModTime().Before(cutoff) {
				// Сжимаем (п.7.2)
				logPath := filepath.Join(logDir, f.Name())
				gzPath := logPath + ".gz"
				
				err := compressFile(logPath, gzPath)
				if err != nil {
					Log.Errorf("Ошибка сжатия лога %s: %v", logPath, err)
				} else {
					Log.Infof("Лог %s сжат в %s", f.Name(), gzPath)
					_ = os.Remove(logPath) // Удаляем оригинал после сжатия
				}
			}
		}
	}
}

func compressFile(src, dst string) error {
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

	gw := gzip.NewWriter(outFile)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	info, err := inFile.Stat()
	if err != nil {
		return err
	}

	header, err := tar.FileInfoHeader(info, info.Name())
	if err != nil {
		return err
	}

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	_, err = io.Copy(tw, inFile)
	return err
}
