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
		flagMsg       string
		showHelp      bool
		showVersion   bool
		runDiagnose   bool
		createCacheDb bool
		removeCacheDb bool
		createPgDb    bool
		installSrv    bool
		uninstallSrv  bool
	)

	// Отображение справки
	flagMsg = "Отображение списка параметров командной строки"
	flag.BoolVar(&showHelp, "help", false, flagMsg)
	flag.BoolVar(&showHelp, "h", false, flagMsg)

	// Отображение версии программы
	flagMsg = "Отображение версии программы"
	flag.BoolVar(&showVersion, "version", false, flagMsg)
	flag.BoolVar(&showVersion, "v", false, flagMsg)

	// Запуск диагностики
	flagMsg = "Запуск диагностики"
	flag.BoolVar(&runDiagnose, "diagnose", false, flagMsg)
	flag.BoolVar(&runDiagnose, "d", false, flagMsg)

	// Создание контейнера с кэширующей СУБД
	flagMsg = "Создание контейнера с кэширующей СУБД"
	flag.BoolVar(&createCacheDb, "create-cacheDb", false, flagMsg)
	flag.BoolVar(&createCacheDb, "c", false, flagMsg)

	// Удаление контейнера с кэширующей СУБД
	flagMsg = "Удаление контейнера с кэширующей СУБД"
	flag.BoolVar(&removeCacheDb, "remove-cacheDb", false, flagMsg)
	flag.BoolVar(&removeCacheDb, "r", false, flagMsg)

	// Создание базы в СУБД PostgreSQL
	flagMsg = "Создание базы в СУБД PostgreSQL"
	flag.BoolVar(&createPgDb, "create-pg-db", false, flagMsg)
	flag.BoolVar(&createPgDb, "p", false, flagMsg)

	// Установка службы Windows
	flagMsg = "Установка службы Windows"
	flag.BoolVar(&installSrv, "install-srv", false, flagMsg)
	flag.BoolVar(&installSrv, "i", false, flagMsg)

	// Удаление службы Windows
	flagMsg = "Удаление службы Windows"
	flag.BoolVar(&uninstallSrv, "uninstall-srv", false, flagMsg)
	flag.BoolVar(&uninstallSrv, "u", false, flagMsg)

	// Настройка вывода справки
	hlpStr := "Venera — Система сбора идентификаторов в потоке пакетных данных\n\n" +
		"Использование:\n" +
		"  venera.exe [опция]\n\n" +
		"Опции:\n"	
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, hlpStr)

		// Карта для группировки флагов по их описанию (Usage)
		groupedFlags := make(map[string][]string)
		
		// Перебираем все зарегистрированные флаги
		flag.VisitAll(func(f *flag.Flag) {
			groupedFlags[f.Usage] = append(groupedFlags[f.Usage], f.Name)
		})

		// Выводим сгруппированные флаги
		for usage, names := range groupedFlags {
			// Формируем строку флагов: добавляем дефисы и соединяем через запятую
			var formattedNames []string
			for _, name := range names {
				if len(name) == 1 {
					formattedNames = append(formattedNames, "-"+name) // Короткий флаг
				} else {
					formattedNames = append(formattedNames, "--"+name) // Длинный флаг
				}
			}
			
			// Соединяем флаги через запятую
			flagsString := strings.Join(formattedNames, ", ")
			
			// Печатаем в консоль с красивым выравниванием (\t — табуляция)
			fmt.Fprintf(os.Stderr, "  %-25s %s\n", flagsString, usage)
		}
	}

	flag.Parse()

	if showHelp {
		flag.Usage()
		os.Exit(0)
	}

	if showVersion {
		fmt.Printf("Venera версия %s\n", version)
		os.Exit(0)
	}

	// Загрузка конфигурации
	if err := config.LoadConfig(); err != nil {
		fmt.Printf("Ошибка загрузки конфигурации: %v\n", err)
	}
	cfg := config.GetConfig()

	// Скрываем консоль сразу, если режим tray и отключено отображение консоли при старте
	if cfg.Generic.Mode == "tray" && !cfg.Generic.ShowConsoleOnStartup {
		utils.HideConsole()
	}

	// Инициализация логгера
	if err := logging.InitLogger(); err != nil {
		fmt.Printf("Критическая ошибка инициализации логгера: %v\n", err)
		os.Exit(1)
	}

	// Обработка CLI команд управления
	handleCLICommands(
		runDiagnose, createCacheDb, removeCacheDb, createPgDb,
		installSrv, uninstallSrv, cfg,
	)

	// Проверка прав администратора при запуске приложения
	if !utils.IsAdmin() {
		// Ограничиваемся только логом, так как вывод в GUI реализован внутри tray.RunTray
		logging.Log.Warn("Внимание: приложение запущено без прав Администратора.")
	}

	// Проверка наличия и запуск Podman
	// В рамках основной логики запускаем кэш БД
	setupCacheDbContainer(cfg.Paths)

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
	if err := data.InitCacheDbConn(); err != nil {
		logging.Log.Errorf("Ошибка подключения к кэширующей СУБД: %v", err)
		tray.ShowErrorNotification("Нет связи с кэширующей СУБД")
	}

	// Подключение к итоговой СУБД
	if err := data.InitPgConn(); err != nil {
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
	data.CloseCacheDbConn()
	data.ClosePgConn()
	logging.Log.Infof("Venera успешно остановлена.")
}

// handleCLICommands обрабатывает эксклюзивные CLI команды
func handleCLICommands(
	runDiagnose, createCacheDb, removeCacheDb, createPgDb,
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

	if createCacheDb {
		setupDragonflyContainer(cfg.Paths)
		fmt.Println("Контейнер кэширующей СУБД успешно проверен/создан.")
		os.Exit(0)
	}

	if removeCacheDb {
		fmt.Println("Удаление контейнера кэширующей СУБД...")
		_ = exec.Command(cfg.Paths.PodmanExe, "rm", "-f", "CacheDb").Run()
		os.Exit(0)
	}

	if createPgDb {
		fmt.Println("Инициализация базы в СУБД PostgreSQL...")
		err := data.InitPGDatabase(&cfg.PostgreSQL)
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

// setupCacheDbContainer проверяет и поднимает контейнер с кэширующей СУБД
func setupCacheDbContainer(paths models.PathsConfig) {
	podman := paths.PodmanExe

	// Проверка наличия podman
 	if _, err := exec.LookPath(podman); err != nil {
 		logging.Log.Warnf("Podman не найден по пути: %s", podman)
 		tray.ShowErrorNotification("Ошибка запуска кэширующей СУБД")
		return
 	}

	// Запуск podman machine (если требуется на Windows)
	if _, err := exec.Command(podman, "machine", "start").Run(); err != nil {
		logging.Log.Errorf("Ошибка запуска Podman: %v", err)
		tray.ShowErrorNotification("Ошибка запуска кэширующей СУБД")
		return
	}

	// Проверка наличия контейнера CacheDb
	out, _ := exec.Command(podman, "ps", "-a", "--format", "{{.Names}}").Output()
	if !strings.Contains(string(out), "cachedb") {
		logging.Log.Warnf("Контейнер cachedb не найден")
		logging.Log.Infof("Создание контейнера cachedb из образа %s...", paths.DbImage)
		err := exec.Command("podman", "load", "-i", paths.DbImage).Run()
		err := exec.Command(podman, "run", "-d", "--name", "cachedb", "-p", "6379:6379", paths.DbImage).Run()
		if err != nil {
			logging.Log.Errorf("Ошибка создания контейнера: %v", err)
			tray.ShowErrorNotification("Ошибка запуска кэширующей СУБД")
		}
	} else {
		// Запуск контейнера
		if _, err := exec.Command(podman, "start", "cachedb").Run(); err != nil {
			logging.Log.Errorf("Ошибка запуска контейнера: %v", err)
			tray.ShowErrorNotification("Ошибка запуска кэширующей СУБД")
		}
	}
}
