package diagnose_test

import (
	"os"
	"testing"
	"venera/config"
	"venera/diagnose"
	"venera/logging"

	"github.com/sirupsen/logrus"
)

func init() {
	logging.Log = logrus.New()
	logging.Log.SetOutput(os.Stdout)
	// Инициализация конфига для избежания nil pointer
	config.GlobalConfig = config.DefaultConfig()
}

func TestDiagnoseReport(t *testing.T) {
	// 1. Получение отчета
	report, err := diagnose.RunDiagnosis()
	if err != nil {
		t.Fatalf("Ошибка генерации отчета: %v", err)
	}

	if report == nil {
		t.Fatal("Отчет пуст")
	}

	// 2. Экспорт в PDF
	pdfPath := "test_report.pdf"
	err = diagnose.ExportReportPDF(report, pdfPath)
	if err != nil {
		t.Fatalf("Ошибка экспорта в PDF: %v", err)
	}
	defer os.Remove(pdfPath)

	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Error("Файл PDF не был создан")
	}

	// 3. Создание архива
	gzPath := "test_archive.tar.gz"
	err = diagnose.CreateArchiveGZ(pdfPath, gzPath)
	if err != nil {
		t.Fatalf("Ошибка создания архива: %v", err)
	}
	defer os.Remove(gzPath)

	if _, err := os.Stat(gzPath); os.IsNotExist(err) {
		t.Error("Файл GZ не был создан")
	}
}
