package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"venera/config"
	"venera/data"
	"venera/diagnose"
	"venera/filter"
	"venera/logging"
	"venera/processes"
	"venera/services"
	"venera/tray"
	"venera/web"
)

var (
	version = "1.0.0"
)

func main() {
	showHelp := flag.Bool("help", false, "Показать справку")
	showHelpShort := flag.Bool("h", false, "Показать справку")
	showVersion := flag.Bool("version", false, "Показать версию программы")
	showVersionShort := flag.Bool("v", false, "Показать версию программы")
	runDiagnose := flag.Bool("diagnose", false, "Запустить диагностику системы")
	runDiagnoseShort := flag.Bool("d", false, "Запустить диагностику системы")
	
	flag.Parse()

	if *showHelp || *showHelpShort {
		printHelp()
		os.Exit(0)
	}

	if *showVersion || *showVersionShort {
		fmt.Printf("Venera Version: %s\n", version)
		os.Exit(0)
	}

	// Загрузка конфигурации
	if err := config.LoadConfig(); err != nil {
		fmt.Printf("Ошибка загрузки конфигурации: %v\n", err)
	}

	// Инициализация логгера
	if err := logging.InitLogger(); err != nil {
		fmt.Printf("Критическая ошибка инициализации логгера: %v\n", err)
		os.Exit(1)
	}

	if *runDiagnose || *runDiagnoseShort {
		diagnose.RunConsoleDiagnosis()
		os.Exit(0)
	}

	// Загрузка процессов
	if err := processes.LoadProcesses(); err != nil {
		logging.Log.Warnf("Ошибка загрузки списка процессов: %v", err)
	}

	// Загрузка списков фильтрации и контроля (без блокировок)
	_ = filter.LoadFilterList()
	_ = filter.LoadControlList()

	mode := config.GlobalConfig.Generic.Mode

	if mode == "service" {
		err := services.RunService(startApplication, stopApplication)
		if err != nil {
			logging.Log.Fatalf("Ошибка запуска службы: %v", err)
		}
	} else {
		// Запуск в Tray
		tray.RunTray(startApplication, stopApplication)
	}
}

func startApplication() {
	if err := data.InitDragonflyDB(); err != nil {
		logging.Log.Errorf("Ошибка БД Dragonfly: %v", err)
	}

	if err := data.InitPostgreSQL(); err != nil {
		logging.Log.Errorf("Ошибка БД PostgreSQL: %v", err)
	}

	if config.GlobalConfig.Generic.AutoStart {
		for _, p := range processes.GetAllProcesses() {
			if err := processes.Manager.StartProcess(p); err != nil {
				logging.Log.Errorf("Ошибка автостарта процесса %s: %v", p.ID, err)
			}
		}
	}

	web.StartWebServer()

	// Graceful shutdown по сигналам OS
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs
		logging.Log.Infof("Получен сигнал завершения. Остановка...")
		stopApplication()
		os.Exit(0)
	}()
}

func stopApplication() {
	web.StopWebServer()
	
	// Безопасная остановка всех запущенных процессов Tshark (Job Objects) и Горутин
	for _, p := range processes.GetAllProcesses() {
		if p.Status == "running" {
			_ = processes.Manager.StopProcess(p.ID)
		}
	}
	
	data.CloseDragonflyDB()
	data.ClosePostgreSQL()
	
	logging.Log.Infof("Venera успешно остановлена.")
}

func printHelp() {
	fmt.Println("Venera — Система сбора идентификаторов в потоке пакетных данных")
	fmt.Println("\nИспользование:")
	fmt.Println("  venera.exe [опция]")
	fmt.Println("\nОпции:")
	flag.PrintDefaults()
}