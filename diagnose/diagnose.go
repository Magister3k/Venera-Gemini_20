package diagnose

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"venera/config"
	"venera/logging"
)

// RunConsoleDiagnosis запускает модуль диагностики
func RunConsoleDiagnosis() {
	fmt.Println("=== Диагностика системы Venera ===")
	cfg := config.GlobalConfig

	// 1. Проверка прав на запись
	logDir := filepath.Dir(config.GlobalConfig.Paths.FilterList) // Используем settings папку
	fmt.Printf("Проверка прав на запись в директорию %s... ", logDir)
	testFile := filepath.Join(logDir, ".test_write")
	err := os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		fmt.Printf("[ОШИБКА] (%v)\n", err)
	} else {
		os.Remove(testFile)
		fmt.Println("[ОК]")
	}

	// 2. Проверка Tshark
	fmt.Printf("Проверка исполняемого файла Tshark (%s)... ", cfg.Paths.TsharkExe)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, cfg.Paths.TsharkExe, "-v")
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("[ОШИБКА] (%v)\n", err)
	} else {
		// Выводим только первую строку версии
		firstLine := ""
		for _, c := range out {
			if c == '\n' {
				break
			}
			firstLine += string(c)
		}
		fmt.Printf("[ОК] (%s)\n", firstLine)
	}

	// 3. Проверка портов баз данных
	fmt.Printf("Проверка порта DragonflyDB (%s:%d)... ", cfg.DragonflyDB.Host, cfg.DragonflyDB.Port)
	dfAddr := fmt.Sprintf("%s:%d", cfg.DragonflyDB.Host, cfg.DragonflyDB.Port)
	connDF, err := net.DialTimeout("tcp", dfAddr, 2*time.Second)
	if err != nil {
		fmt.Printf("[ОШИБКА] (%v)\n", err)
	} else {
		connDF.Close()
		fmt.Println("[ОК]")
	}

	fmt.Printf("Проверка порта PostgreSQL (%s:%d)... ", cfg.PostgreSQL.Host, cfg.PostgreSQL.Port)
	pgAddr := fmt.Sprintf("%s:%d", cfg.PostgreSQL.Host, cfg.PostgreSQL.Port)
	connPG, err := net.DialTimeout("tcp", pgAddr, 2*time.Second)
	if err != nil {
		fmt.Printf("[ОШИБКА] (%v)\n", err)
	} else {
		connPG.Close()
		fmt.Println("[ОК]")
	}
	
	logging.Log.Info("Диагностика завершена")
	fmt.Println("=== Диагностика завершена ===")
}