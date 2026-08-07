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
	"venera/containers"
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
	"venera/utils"
	"venera/web"
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
		createCacheDb bool
		removeCacheDb bool
		createPgDb    bool
		installSvc    bool
		uninstallSvc  bool
	)

	// Отображение справки
	flagMsg := "Отображение списка параметров командной строки"
	flag.BoolVar(&showHelp, "help", false, flagMsg)
	flag.BoolVar(&showHelp, "h", false, flagMsg)

	// Отображение версии программы
	flagMsg := "Отображение версии программы"
	flag.BoolVar(&showVersion, "version", false, flagMsg)
	flag.BoolVar(&showVersion, "v", false, flagMsg)

	// Запуск диагностики
	flagMsg := "Запуск диагностики"
	flag.BoolVar(&runDiagnose, "diagnose", false, flagMsg)
	flag.BoolVar(&runDiagnose, "d", false, flagMsg)

	// Создание контейнера с кэширующей СУБД
	flagMsg := "Создание контейнера с кэширующей СУБД"
	flag.BoolVar(&createCacheDb, "create-cacheDb", false, flagMsg)
	flag.BoolVar(&createCacheDb, "c", false, flagMsg)

	// Удаление контейнера с кэширующей СУБД
	flagMsg := "Удаление контейнера с кэширующей СУБД"
	flag.BoolVar(&removeCacheDb, "remove-cacheDb", false, flagMsg)
	flag.BoolVar(&removeCacheDb, "r", false, flagMsg)

	// Создание базы в СУБД PostgreSQL
	flagMsg := "Создание базы в СУБД PostgreSQL"
	flag.BoolVar(&createPgDb, "create-pg-db", false, flagMsg)
	flag.BoolVar(&createPgDb, "p", false, flagMsg)

	// Установка службы Windows
	flagMsg := "Установка службы Windows"
	flag.BoolVar(&installSvc, "install-svc", false, flagMsg)
	flag.BoolVar(&installSvc, "i", false, flagMsg)

	// Удаление службы Windows
	flagMsg := "Удаление службы Windows"
	flag.BoolVar(&uninstallSvc, "uninstall-svc", false, flagMsg)
	flag.BoolVar(&uninstallSvc, "u", false, flagMsg)

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
			flagsStr := strings.Join(formattedNames, ", ")
			
			// Печатаем в консоль с красивым выравниванием (\t — табуляция)
			fmt.Fprintf(os.Stderr, "  %-25s %s\n", flagsStr, usage)
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
	if err := config.LoadCfg(); err != nil {
		fmt.Printf("Ошибка загрузки конфигурации: %v\n", err)
	}
	cfg := config.GetCfg()

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
	handleCLICmds(
		runDiagnose, createCacheDb, removeCacheDb, createPgDb,
		installSvc, uninstallSvc, cfg
	)

	// Проверка прав администратора при запуске приложения
	if !utils.IsAdmin() {
		// Ограничиваемся только логом, так как вывод в GUI реализован внутри tray.RunTray
		logging.Log.Warn("Внимание: приложение запущено без прав Администратора.")
	}

	// Запускаем кэширующую СУБД
	containers.startCacheDb(cfg.Paths.Podman, cfg.Paths.CacheDbImage)

	// Загрузка списков
	_ = data.LoadFilter(cfg.Paths.Filter)
	_ = data.LoadControl(cfg.Paths.Control)
	_ = notify.LoadAlerts(cfg.Paths.Alerts)

	// Загрузка процессов
	if err := processes.LoadProcs(); err != nil {
		logging.Log.Warnf("Ошибка загрузки списка процессов: %v", err)
	}

	// Регистрация и проверка манифеста
	if err := manifest.Init(); err != nil {
		logging.Log.Errorf("Ошибка инициализации манифеста ETW: %v", err)
	}

	// Выбор режима запуска
	if cfg.Generic.Mode == "service" {
		err := services.RunSvc(startApplication, stopApplication)
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
		utils.ShowBalloonNotify("Venera", "Нет связи с кэширующей СУБД")
	}

	// Подключение к итоговой базе в СУБД PostgreSQL
	if err := data.InitPgConn(); err != nil {
		logging.Log.Errorf("Ошибка подключения к итоговой базе в СУБД PostgreSQL: %v", err)
		utils.ShowBalloonNotify("Venera", "Нет связи с итоговой базой в СУБД PostgreSQL")
	}

	// Запуск фонового мониторинга и защиты
	monitorCtx, monitorCancel := context.WithCancel(context.Background())
	metrics.SetStopAllCallback(stopAllProcs)
	go metrics.StartMonitor(monitorCtx)

	// Запуск процессов, если включен автостарт
	cfg := config.GetCfg()
	if cfg.Generic.AutoStart {
		for _, p := range processes.GetAllProcs() {
			if err := processes.Manager.StartProc(p); err != nil {
				logging.Log.Errorf("Ошибка автостарта процесса %s: %v", p.ID, err)
			}
		}
	}

	// Старт веб-интерфейса
	go web.StartWebSrv()

	// Graceful shutdown по сигналам Windows
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs
		logging.Log.Infof("Получен сигнал завершения. Остановка...")
		monitorCancel()
		stopApp()
		os.Exit(0)
	}()
}

func stopAllProcs() {
	for _, p := range processes.GetAllProcs() {
		if string(p.Status) == "running" {
			_ = processes.Manager.StopProc(p.ID)
		}
	}
}

func stopApp() {
	web.StopWebSrv()
	stopAllProcs()
	data.CloseCacheDbConn()
	data.ClosePgConn()
	logging.Log.Infof("Venera успешно остановлена.")
}

// handleCLICmds обрабатывает эксклюзивные CLI команды
func handleCLICmds(
	runDiagnose, createCacheDb, removeCacheDb, createPgDb,
	installSvc, uninstallSvc bool, cfg models.Config) {

	if runDiagnose {
		fmt.Println("Запуск полной диагностики системы...")
		report, err := diagnose.RunDiag()
		if err != nil {
			fmt.Printf("Критическая ошибка диагностики: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Диагностика завершена. Проверка RAM: %v, СУБД: %v\n", report.FreeRAMBytes, report.PgConnected)

		pdfPath := "Venera_Diagnostic_Report.pdf"
		_ = diagnose.ExportReportPDF(report, pdfPath)

		gzPath := "Venera_Diagnostic_Archive.tar.gz"
		_ = diagnose.CreateArchiveGZ(pdfPath, gzPath)

		fmt.Printf("Отчеты сохранены:\n- %s\n- %s\n", pdfPath, gzPath)
		os.Exit(0)
	}

	if createCacheDb {
		fmt.Println("Создание контейнера с кэширующей СУБД...")
		containers.startCacheDb(cfg.Paths.Podman, cfg.Paths.CacheDbImage)
		if err != nil {
			fmt.Printf("Ошибка создания контейнера с кэширующей СУБД: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Контейнер с кэширующей СУБД успешно создан.")
		os.Exit(0)
	}

	if removeCacheDb {
		fmt.Println("Удаление контейнера с кэширующей СУБД...")
		_ = exec.Command(cfg.Paths.Podman, "rm", "-f", "cachedb").Run()
		if err != nil {
			fmt.Printf("Ошибка удаления контейнера с кэширующей СУБД: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Контейнер с кэширующей СУБД успешно удален.")
		os.Exit(0)
	}

	if createPgDb {
		fmt.Println("Создание итоговой базы в СУБД PostgreSQL...")
		err := data.InitPgDb(&cfg.PostgreSQL)
		if err != nil {
			fmt.Printf("Ошибка создания итоговой базы в СУБД PostgreSQL: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Итоговая база данных успешно создана.")
		os.Exit(0)
	}

	if installSvc {
		if err := services.InstallSvc(); err != nil {
			fmt.Printf("Ошибка установки службы Windows: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if uninstallSvc {
		if err := services.UninstallSvc(); err != nil {
			fmt.Printf("Ошибка удаления службы Windows: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}
}
