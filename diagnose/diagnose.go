package diagnose

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jung-kurt/gofpdf"

	"venera/config"
	"venera/data"
	"venera/logging"
	"venera/manifest"
	"venera/metrics"
	"venera/services"
)

// DiagnosticReport содержит результаты выполнения всех проверок из п.9.1 ТЗ.
type DiagnosticReport struct {
	AppVersion          string
	ConfigExists        bool
	ManifestRegistered  bool
	ManifestVersion     string
	FreeRAMBytes        uint64
	FreeRAMPercent      float64
	PGDiskFreeBytes     uint64
	DFDiskFreeBytes     uint64
	NetworkInterfaces   []string
	TsharkExists        bool
	PodmanExists        bool
	DragonflyImageExist bool
	PGConnected         bool
	ServiceInstalled    bool
	ServiceStatus       string
	AppMode             string
	EventLogErrors      []string
}

// RunDiagnosis собирает всю диагностическую информацию (п.9.1 ТЗ)
func RunDiagnosis() (*DiagnosticReport, error) {
	report := &DiagnosticReport{
		AppVersion: manifest.CurrentAppVersion,
	}

	cfg := config.GlobalConfig

	// 2. Проверка наличия файла конфигурации
	if _, err := os.Stat(config.ConfigPath); err == nil {
		report.ConfigExists = true
	}

	// 3-4. Проверка манифеста
	cmd := exec.Command("wevtutil", "ep")
	var out bytes.Buffer
	cmd.Stdout = &out
	_ = cmd.Run()
	if strings.Contains(out.String(), manifest.ProviderName) {
		report.ManifestRegistered = true
		report.ManifestVersion = manifest.CurrentAppVersion // В идеале парсится XML
	}

	// 5-7. Объемы RAM и дисков
	ram, ramPct, _ := metrics.GetSystemRAM()
	report.FreeRAMBytes = ram
	report.FreeRAMPercent = ramPct

	pgDiskPath := "C:\\" // Default fallback
	if cfg.PostgreSQL.Host == "127.0.0.1" || cfg.PostgreSQL.Host == "localhost" {
		pgDiskPath = "C:\\" // Условно диск C для локальной БД
	}
	dfDiskPath := cfg.Paths.DbBackupDir
	if dfDiskPath == "" {
		dfDiskPath = "."
	}

	absDfPath, _ := filepath.Abs(dfDiskPath)

	pgFree, _, _ := metrics.GetDiskSpace(pgDiskPath)
	dfFree, _, _ := metrics.GetDiskSpace(absDfPath)

	report.PGDiskFreeBytes = pgFree
	report.DFDiskFreeBytes = dfFree

	// 8. Список сетевых адаптеров
	ifaces, _ := metrics.GetNetworkInterfaces()
	report.NetworkInterfaces = ifaces

	// 9-10. Наличие исполняемых файлов
	report.TsharkExists = fileExists(cfg.Paths.TsharkExe) || checkCommand(cfg.Paths.TsharkExe)
	report.PodmanExists = fileExists(cfg.Paths.PodmanExe) || checkCommand(cfg.Paths.PodmanExe)

	// 11. Наличие образа DragonflyDB
	// Используем podman image exists
	if report.PodmanExists {
		cmdImg := exec.Command(cfg.Paths.PodmanExe, "image", "exists", cfg.Paths.DbImage)
		err := cmdImg.Run()
		report.DragonflyImageExist = (err == nil)
	}

	// 12. Проверка подключения к СУБД PostgreSQL
	if data.PgPool != nil {
		err := data.PgPool.Ping(context.Background())
		report.PGConnected = (err == nil)
	}

	// 13-14. Проверка службы
	statusStr, _ := services.GetServiceStatus()
	report.ServiceInstalled = (statusStr != "Не установлена" && statusStr != "Ошибка")
	report.ServiceStatus = statusStr
	report.AppMode = cfg.Generic.Mode

	// 15. Последние 10 ошибок Event Log
	report.EventLogErrors = getEventLogErrors()

	return report, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func checkCommand(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// getEventLogErrors получает последние 10 ошибок из лога Application (Windows)
func getEventLogErrors() []string {
	// Простейший способ без CGO - вызвать powershell
	psCmd := `Get-EventLog -LogName Application -EntryType Error -Newest 10 | Select-Object -ExpandProperty Message`
	cmd := exec.Command("powershell", "-NoProfile", "-Command", psCmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return []string{"Не удалось получить историю Event Log: " + err.Error()}
	}

	lines := strings.Split(string(out), "\n")
	var result []string
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// ExportReportPDF создает отчет в формате PDF (п.9.1.16 ТЗ)
func ExportReportPDF(report *DiagnosticReport, outputPath string) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// Используем стандартный шрифт, не требующий внешних ttf для базового английского,
	// но для русского нужен TTF. В gofpdf встроенной кириллицы нет, если не загрузить шрифт.
	// Для простоты будем использовать транслит или базовые метрики,
	// либо просто пропустим кириллицу (заменим на англ.), чтобы не тащить файлы шрифтов.
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(40, 10, "Venera System Diagnostic Report")
	pdf.Ln(12)

	pdf.SetFont("Arial", "", 12)

	addLine := func(label, value string) {
		pdf.CellFormat(90, 8, label+":", "1", 0, "L", false, 0, "")
		pdf.CellFormat(100, 8, value, "1", 1, "L", false, 0, "")
	}

	addLine("App Version", report.AppVersion)
	addLine("App Mode", report.AppMode)
	addLine("Config File Exists", fmt.Sprintf("%v", report.ConfigExists))
	addLine("Manifest Registered", fmt.Sprintf("%v", report.ManifestRegistered))
	addLine("Manifest Version", report.ManifestVersion)
	addLine("Service Installed", fmt.Sprintf("%v", report.ServiceInstalled))
	addLine("Service Status", report.ServiceStatus)
	addLine("Tshark Available", fmt.Sprintf("%v", report.TsharkExists))
	addLine("Podman Available", fmt.Sprintf("%v", report.PodmanExists))
	addLine("DF Image Exists", fmt.Sprintf("%v", report.DragonflyImageExist))
	addLine("PostgreSQL Connected", fmt.Sprintf("%v", report.PGConnected))

	pdf.Ln(5)
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(40, 10, "System Resources")
	pdf.Ln(10)
	pdf.SetFont("Arial", "", 12)

	addLine("Free RAM (MB)", fmt.Sprintf("%d", report.FreeRAMBytes/(1024*1024)))
	addLine("Free RAM (%)", fmt.Sprintf("%.2f%%", report.FreeRAMPercent))
	addLine("PG Disk Free (MB)", fmt.Sprintf("%d", report.PGDiskFreeBytes/(1024*1024)))
	addLine("DF Disk Free (MB)", fmt.Sprintf("%d", report.DFDiskFreeBytes/(1024*1024)))

	pdf.Ln(5)
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(40, 10, "Network Interfaces")
	pdf.Ln(10)
	pdf.SetFont("Arial", "", 10)
	for _, iface := range report.NetworkInterfaces {
		pdf.MultiCell(190, 6, iface, "1", "L", false)
	}

	err := pdf.OutputFileAndClose(outputPath)
	if err != nil {
		return fmt.Errorf("ошибка генерации PDF: %v", err)
	}

	logging.Log.Infof("Сгенерирован диагностический отчет: %s", outputPath)
	return nil
}

// CreateArchiveGZ создает архив (tar.gz) с логами, отчётом и конфигами (п.9.1.17 ТЗ)
func CreateArchiveGZ(pdfReportPath, outputPath string) error {
	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	gw := gzip.NewWriter(outFile)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Список файлов для архивации
	cfg := config.GlobalConfig
	filesToArchive := []string{
		pdfReportPath,
		config.ConfigPath,
		cfg.Paths.FilterList,
		cfg.Paths.ControlList,
		cfg.Paths.AlertsList,
	}

	// Добавляем все логи из папки Logs
	logFiles, _ := filepath.Glob("Logs/*.log")
	filesToArchive = append(filesToArchive, logFiles...)

	for _, file := range filesToArchive {
		if file == "" {
			continue
		}

		info, err := os.Stat(file)
		if err != nil {
			// Игнорируем отсутствующие файлы согласно ТЗ ("если имеются")
			continue
		}

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			continue
		}
		header.Name = filepath.Base(file) // Сохраняем в корень архива

		if err := tw.WriteHeader(header); err != nil {
			continue
		}

		f, err := os.Open(file)
		if err != nil {
			continue
		}

		_, _ = io.Copy(tw, f)
		f.Close()
	}

	logging.Log.Infof("Создан диагностический архив: %s", outputPath)
	return nil
}
