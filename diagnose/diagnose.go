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

	"github.com/signintech/gopdf"

	"venera/config"
	"venera/data"
	"venera/logging"
	"venera/manifest"
	"venera/metrics"
	"venera/services"
)

// DiagReport содержит результаты выполнения всех проверок.
type DiagReport struct {
	AppVersion            string
	CfgExists             bool
	ManifestRegistered    bool
	ManifestVersion       string
	FreeRAMBytes          uint64
	FreeRAMPerc           float64
	TargetDbDiskFreeBytes uint64
	CacheDbDiskFreeBytes  uint64
	NetInterfaces         []string
	TsharkExists          bool
	PodmanExists          bool
	CacheDbImageExist     bool
	PgConnected           bool
	SvcInstalled          bool
	SvcStatus             string
	AppMode               string
	EventLogErrors        []string
}

// RunDiag собирает всю диагностическую информацию
func RunDiag() (*DiagReport, error) {
	report := &DiagReport{
		AppVersion: manifest.CurrentAppVersion,
	}

	cfg := config.GlobalCfg

	// Проверка наличия файла конфигурации
	if _, err := os.Stat(config.CfgPath); err == nil {
		report.CfgExists = true
	}

	// Проверка манифеста
	cmd := exec.Command("wevtutil", "ep")
	var out bytes.Buffer
	cmd.Stdout = &out
	_ = cmd.Run()
	if strings.Contains(out.String(), manifest.ProviderName) {
		report.ManifestRegistered = true
		report.ManifestVersion = manifest.CurrentAppVersion // В идеале парсится XML
	}

	// Объемы RAM и дисков
	ram, ramPct, _ := metrics.GetSysRAM()
	report.FreeRAMBytes = ram
	report.FreeRAMPerc = ramPct

	// TODO: определять путь к диску с итоговой базой средствами СУБД PostgreSQL
	pgDiskPath := "C:\\" // Default fallback
	if cfg.PostgreSQL.Host == "127.0.0.1" || cfg.PostgreSQL.Host == "localhost" {
		pgDiskPath = "C:\\" // Условно диск C для локальной БД
	}
	cacheDbDiskPath := cfg.Paths.CacheDir
	if cacheDbDiskPath == "" {
		cacheDbDiskPath = "."
	}

	absCacheDbPath, _ := filepath.Abs(cacheDbDiskPath)

	pgFree, _, _ := metrics.GetDiskSpace(pgDiskPath)
	cacheDbFree, _, _ := metrics.GetDiskSpace(absCacheDbPath)

	report.TargetDbDiskFreeBytes = pgFree
	report.CacheDbDiskFreeBytes = cacheDbFree

	// Список сетевых адаптеров
	ifaces, _ := metrics.GetNetInterfaces()
	report.NetworkInterfaces = ifaces

	// Наличие исполняемых файлов
	report.TsharkExists = fileExists(cfg.Paths.Tshark) || checkCommand(cfg.Paths.Tshark)
	report.PodmanExists = fileExists(cfg.Paths.Podman) || checkCommand(cfg.Paths.Podman)

	// Наличие локального файла с образом кэширующей СУБД
	// Используем podman image exists
	if report.PodmanExists {
		err := exec.Command(cfg.Paths.Podman, "image", "exists", cfg.Paths.DbImage).Run()
		report.CacheDbImageExist = (err == nil)
	}

	// Проверка подключения к итоговой базе в СУБД PostgreSQL
	if data.PgPool != nil {
		err := data.PgPool.Ping(context.Background())
		report.PgConnected = (err == nil)
	}

	// Проверка службы
	statusStr, _ := services.GetSvcStatus()
	report.SvcInstalled = (statusStr != "Не установлена" && statusStr != "Ошибка")
	report.SvcStatus = statusStr
	report.AppMode = cfg.Generic.Mode

	// Последние 10 ошибок Event Log
	report.EventLogErrors = getEventLogErrors()

	return report, nil
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func CheckCmd(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// GetLastEventLogErrors получает последние 10 ошибок из лога Application в Windows
func GetLastEventLogErrors() []string {
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

// ExportReportPDF создает отчет в формате PDF на русском языке
func ExportReportPDF(report *DiagReport, outputPath string) error {
	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	pdf.AddPage()

	// Добавляем поддержку кириллицы через шрифт (требуется ttf файл)
	// В Windows стандартный шрифт Arial обычно лежит в C:\Windows\Fonts\arial.ttf
	fontPath := "C:\\Windows\\Fonts\\arial.ttf"
	err := pdf.AddTTFFont("Arial", fontPath)
	if err != nil {
		logging.Log.Warnf("Шрифт Arial не найден: %v. PDF может не отображать кириллицу.", err)
		// Fallback без кириллицы (может выдать кракозябры или ошибки, если шрифт не загружен, но gopdf требует TTF для unicode)
	}

	err = pdf.SetFont("Arial", "", 16)
	if err != nil {
		return fmt.Errorf("ошибка установки шрифта: %v", err)
	}

	pdf.Cell(nil, "Отчет о диагностике системы Venera")
	pdf.Br(20)

	pdf.SetFont("Arial", "", 12)

	addLine := func(label, value string) {
		pdf.Cell(nil, fmt.Sprintf("%s: %s", label, value))
		pdf.Br(10)
	}

	addLine("Версия приложения", report.AppVersion)
	addLine("Режим работы", report.AppMode)
	addLine("Файл конфигурации", fmt.Sprintf("%v", report.CfgExists))
	addLine("Манифест зарегистрирован", fmt.Sprintf("%v", report.ManifestRegistered))
	addLine("Версия манифеста", report.ManifestVersion)
	addLine("Служба установлена", fmt.Sprintf("%v", report.SvcInstalled))
	addLine("Статус службы", report.SvcStatus)
	addLine("Доступен Tshark", fmt.Sprintf("%v", report.TsharkExists))
	addLine("Доступен Podman", fmt.Sprintf("%v", report.PodmanExists))
	addLine("Образ DragonflyDB загружен", fmt.Sprintf("%v", report.CacheDbImageExist))
	addLine("СУБД PostgreSQL подключена", fmt.Sprintf("%v", report.PgConnected))

	pdf.Br(10)
	pdf.SetFont("Arial", "", 14)
	pdf.Cell(nil, "Системные ресурсы")
	pdf.Br(15)
	pdf.SetFont("Arial", "", 12)

	addLine("Свободная ОЗУ (MB)", fmt.Sprintf("%d", report.FreeRAMBytes/(1024*1024)))
	addLine("Свободная ОЗУ (%)", fmt.Sprintf("%.2f%%", report.FreeRAMPerc))
	addLine("Свободное место на диске с итоговой БД (MB)", fmt.Sprintf("%d", report.TargetDbDiskFreeBytes/(1024*1024)))
	addLine("Свободное место на диске с бэкапом кэширующей СУБД (MB)", fmt.Sprintf("%d", report.CacheDbDiskFreeBytes/(1024*1024)))

	pdf.Br(10)
	pdf.SetFont("Arial", "", 14)
	pdf.Cell(nil, "Сетевые адаптеры")
	pdf.Br(15)
	pdf.SetFont("Arial", "", 10)
	for _, iface := range report.NetworkInterfaces {
		pdf.Cell(nil, iface)
		pdf.Br(8)
	}

	err = pdf.WritePdf(outputPath)
	if err != nil {
		return fmt.Errorf("ошибка сохранения PDF: %v", err)
	}

	logging.Log.Infof("Сгенерирован диагностический отчет: %s", outputPath)
	return nil
}

// CreateArchGz создает архив (tar.gz) с логами, отчётом и конфигами
func CreateArchGz(pdfReportPath, outputPath string) error {
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
	cfg := config.GlobalCfg
	filesToArchive := []string{
		pdfReportPath,
		config.ConfigPath,
		cfg.Paths.Filter,
		cfg.Paths.Control,
		cfg.Paths.Alerts,
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
