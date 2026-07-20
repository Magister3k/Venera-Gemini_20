package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"venera/config"
	"venera/data"
	"venera/diagnose"
	"venera/logging"
	"venera/manifest"
	"venera/metrics"
	"venera/models"
	"venera/notify"
	"venera/processes"
	"venera/services"
	"venera/sql"
	"venera/tray"
	"venera/web"
	"venera/utils"
)

var (
	// version задается при сборке или используется значение по умолчанию
	version = "1.0.0"
)

func main() {
	// Инициализация флагов
	var (
		showHelp      bool
		showVersion   bool
		runDiagnose   bool
		createCacheDB bool
		removeCacheDB bool
		createPgDB    bool
		installSrv    bool
		uninstallSrv  bool
	)

	// Отображение справки
	flag.BoolVar(&showHelp, "help", false, "Отображение списка параметров командной строки")
	flag.BoolVar(&showHelp, "h", false, "Отображение списка параметров командной строки")

	// Отображение версии программы
	flag.BoolVar(&showVersion, "version", false, "Отображение версии программы")
	flag.BoolVar(&showVersion, "v", false, "Отображение версии программы")

	// Запуск диагностики
	flag.BoolVar(&runDiagnose, "diagnose", false, "Запуск диагностики")
	flag.BoolVar(&runDiagnose, "d", false, "Запуск диагностики")

	// Создание контейнера кэшируюшей СУБД
	flag.BoolVar(&createCacheDB, "create_cachedb", false, "Создание контейнера кэшируюшей СУБД")
	flag.BoolVar(&createCacheDB, "c", false, "Создание контейнера кэшируюшей СУБД")

	// Удаление контейнера кэшируюшей СУБД
	flag.BoolVar(&removeCacheDB, "remove_cachedb", false, "Удаление контейнера кэшируюшей СУБД")
	flag.BoolVar(&removeCacheDB, "r", false, "Удаление контейнера кэшируюшей СУБД")

	// Создание базы в СУБД PostgreSQL
	flag.BoolVar(&createPgDB, "create_pg_db", false, "Создание базы в СУБД PostgreSQL")
	flag.BoolVar(&createPgDB, "p", false, "Создание базы в СУБД PostgreSQL")

	// Установка службы Windows
	flag.BoolVar(&installSrv, "install_srv", false, "Установка службы Windows")
	flag.BoolVar(&installSrv, "i", false, "Установка службы Windows")

	// Удаление службы Windows
	flag.BoolVar(&uninstallSrv, "uninstall_srv", false, "Удаление службы Windows")
	flag.BoolVar(&uninstallSrv, "u", false, "Удаление службы Windows")

	flag.Parse()

	if showHelp {
		printHelp()
		os.Exit(0)
	}

	if showVersion {
		fmt.Printf("Venera Version: %s\n", version)
		os.Exit(0)
	}

	// Загрузка конфигурации
	if err := config.LoadConfig(); err != nil {
		fmt.Printf("Ошибка загрузки конфигурации: %v\n", err)
	}
	cfg := config.GetConfig()

	// Инициализация логгера
	if err := logging.InitLogger(); err != nil {
		fmt.Printf("Критическая ошибка инициализации логгера: %v\n", err)
		os.Exit(1)
	}

	// Обработка CLI команд управления
	handleCLICommands(
		runDiagnose, createCacheDB, removeCacheDB, createPgDB,
		installSrv, uninstallSrv, cfg,
	)

	// Проверка прав администратора при запуске приложения
	if !utils.IsAdmin() {
		// Ограничиваемся только логом, так как вывод в GUI реализован внутри tray.RunTray
		logging.Log.Warn("Внимание: приложение запущено без прав Администратора.")
	}

	// Проверка наличия и запуск Podman
	// В рамках основной логики запускаем кэш БД
	setupDragonflyContainer(cfg.Paths)

	// Загрузка списков
	_ = data.LoadFilters(cfg.Paths.FilterList)
	_ = data.LoadControlList(cfg.Paths.ControlList)
	_ = notify.LoadAlerts(cfg.Paths.AlertsList)

	// Загрузка процессов
	if err := processes.LoadProcesses(); err != nil {
		logging.Log.Warnf("Ошибка загрузки списка процессов: %v", err)
	}

	// Регистрация и проверка манифеста
	if err := manifest.Init(); err != nil {
		logging.Log.Errorf("Ошибка инициализации манифеста ETW: %v", err)
	}

	// Выбор режима запуска
	if cfg.Generic.Mode == "service" {
		err := services.RunService(startApplication, stopApplication)
		if err != nil {
			logging.Log.Fatalf("Ошибка запуска службы: %v", err)
		}
	} else {
		// Запуск в виде приложения системного трея
		tray.RunTray(startApplication, stopApplication)
	}
}

func startApplication() {
	// Подключение к кэширующей СУБД
	if err := data.InitDragonflyDB(); err != nil {
		logging.Log.Errorf("Ошибка подключения к кэширующей СУБД: %v", err)
		tray.ShowErrorNotification("Нет связи с кэширующей СУБД")
	}

	if err := data.InitPostgreSQL(); err != nil {
		logging.Log.Errorf("Ошибка подключения к СУБД PostgreSQL: %v", err)
		tray.ShowErrorNotification("Нет связи с СУБД PostgreSQL")
	}

	// Запуск фонового мониторинга и защиты
	monitorCtx, monitorCancel := context.WithCancel(context.Background())
	metrics.SetStopAllCallback(stopAllProcesses)
	go metrics.StartMonitor(monitorCtx)

	// Запуск процессов, если включен автостарт
	cfg := config.GetConfig()
	if cfg.Generic.AutoStart {
		for _, p := range processes.GetAllProcesses() {
			if err := processes.Manager.StartProcess(p); err != nil {
				logging.Log.Errorf("Ошибка автостарта процесса %s: %v", p.ID, err)
			}
		}
	}

	// Старт веб-интерфейса
	go web.StartWebServer()

	// Graceful shutdown по сигналам Windows
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs
		logging.Log.Infof("Получен сигнал завершения. Остановка...")
		monitorCancel()
		stopApplication()
		os.Exit(0)
	}()
}

func stopAllProcesses() {
	for _, p := range processes.GetAllProcesses() {
		if string(p.Status) == "running" {
			_ = processes.Manager.StopProcess(p.ID)
		}
	}
}

func stopApplication() {
	web.StopWebServer()
	stopAllProcesses()
	data.CloseDragonflyDB()
	data.ClosePostgreSQL()
	logging.Log.Infof("Venera успешно остановлена.")
}

// handleCLICommands обрабатывает эксклюзивные CLI команды
func handleCLICommands(
	runDiagnose, createCacheDB, removeCacheDB, createPgDB,
	installSrv, uninstallSrv bool, cfg models.Config) {

	if runDiagnose {
		fmt.Println("Запуск полной диагностики системы...")
		report, err := diagnose.RunDiagnosis()
		if err != nil {
			fmt.Printf("Критическая ошибка диагностики: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Диагностика завершена. Проверка RAM: %v, СУБД: %v\n", report.FreeRAMBytes, report.PGConnected)

		pdfPath := "Venera_Diagnostic_Report.pdf"
		_ = diagnose.ExportReportPDF(report, pdfPath)

		gzPath := "Venera_Diagnostic_Archive.tar.gz"
		_ = diagnose.CreateArchiveGZ(pdfPath, gzPath)

		fmt.Printf("Отчеты сохранены:\n- %s\n- %s\n", pdfPath, gzPath)
		os.Exit(0)
	}

	if createCacheDB {
		setupDragonflyContainer(cfg.Paths)
		fmt.Println("Контейнер кэширующей СУБД успешно проверен/создан.")
		os.Exit(0)
	}

	if removeCacheDB {
		fmt.Println("Удаление контейнера кэширующей СУБД...")
		_ = exec.Command(cfg.Paths.PodmanExe, "rm", "-f", "cachedb").Run()
		os.Exit(0)
	}

	if createPgDB {
		fmt.Println("Инициализация базы в СУБД PostgreSQL...")
		err := sql.InitializeDatabase(&cfg.PostgreSQL)
		if err != nil {
			fmt.Printf("Ошибка создания базы в СУБД PostgreSQL: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("База данных успешно инициализирована.")
		os.Exit(0)
	}

	if installSrv {
		if err := services.InstallService(); err != nil {
			fmt.Printf("Ошибка установки службы: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if uninstallSrv {
		if err := services.UninstallService(); err != nil {
			fmt.Printf("Ошибка удаления службы: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}
}

// setupDragonflyContainer проверяет и поднимает контейнер с еэширующей СУБД
func setupDragonflyContainer(paths models.PathsConfig) {
	podman := paths.PodmanExe

	// Проверка наличия podman
	if _, err := exec.LookPath(podman); err != nil {
		logging.Log.Warnf("Podman не найден по пути: %s", podman)
		return
	}

	// Запуск podman machine (если требуется на Windows)
	_ = exec.Command(podman, "machine", "start").Run()

	// Проверка наличия контейнера cachedb
	out, _ := exec.Command(podman, "ps", "-a", "--format", "{{.Names}}").Output()
	if !strings.Contains(string(out), "cachedb") {
		logging.Log.Infof("Контейнер cachedb не найден. Создание из образа %s...", paths.DbImage)
		err := exec.Command(podman, "run", "-d", "--name", "cachedb", "-p", "6379:6379", paths.DbImage).Run()
		if err != nil {
			logging.Log.Errorf("Ошибка создания контейнера: %v", err)
			tray.ShowErrorNotification("Ошибка создания БД DragonflyDB")
		}
	} else {
		// Запуск контейнера
		_ = exec.Command(podman, "start", "cachedb").Run()
	}
}

func printHelp() {
	fmt.Println("Venera — Система сбора идентификаторов в потоке пакетных данных")
	fmt.Println("\nИспользование:")
	fmt.Println("  venera.exe [опция]")
	fmt.Println("\nОпции:")
	flag.PrintDefaults()
}
