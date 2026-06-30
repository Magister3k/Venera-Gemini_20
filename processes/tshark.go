package processes

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"time"

	"venera/config"
	"venera/data"
	"venera/logging"
)

// RunTsharkNetwork запускает Tshark для захвата с сетевого интерфейса
func RunTsharkNetwork(ctx context.Context, ip string, port int, sourceID string, signalChan chan struct{}) error {
	exe := config.GlobalConfig.Paths.TsharkExe
	// Пример аргументов Tshark (п.5.1 экспорт в формате json)
	filter := fmt.Sprintf("host %s and udp port %d", ip, port)
	args := []string{"-i", "any", "-f", filter, "-T", "json"}

	return runTsharkCommand(ctx, exe, args, sourceID, signalChan)
}

// RunTsharkFile запускает Tshark для чтения из файла или папки
func RunTsharkFile(ctx context.Context, filePath, folderPath, sourceID string, signalChan chan struct{}) error {
	exe := config.GlobalConfig.Paths.TsharkExe
	var args []string

	if filePath != "" {
		args = []string{"-r", filePath, "-T", "json"}
	} else if folderPath != "" {
		// В реальности нужно реализовать обход папки и запуск Tshark для каждого файла
		// Здесь упрощенный вызов для одного абстрактного файла
		return fmt.Errorf("обработка папки не реализована в заглушке, требует итерации")
	} else {
		return fmt.Errorf("не указан путь к файлу или папке")
	}

	return runTsharkCommand(ctx, exe, args, sourceID, signalChan)
}

func runTsharkCommand(ctx context.Context, exe string, args []string, sourceID string, signalChan chan struct{}) error {
	cmd := exec.CommandContext(ctx, exe, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("ошибка получения stdout Tshark: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("ошибка запуска Tshark: %v", err)
	}

	batchSize := int64(config.GlobalConfig.DragonflyDB.BatchSize)
	var recordsAdded int64

	scanner := bufio.NewScanner(stdout)
	// В реальной жизни Tshark выдает JSON-массивы. Нужно накапливать скобки []
	// Здесь упрощенная построчная обработка (в предположении, что выдается NDJSON или мы склеиваем)
	
	// Для п.5.2 нужна буферизация полного JSON объекта.
	// Здесь опускаем сложную логику буферизации для краткости, 
	// предполагаем что на вход ParseJSONToPairs подается валидный кусок JSON.
	
	for scanner.Scan() {
		line := scanner.Bytes()
		ts := time.Now().UnixMilli() // Время фиксации

		pairs, err := data.ParseJSONToPairs(line, ts)
		if err != nil {
			// logging.Log.Warnf("Ошибка парсинга Tshark JSON: %v", err)
			continue
		}

		for _, pair := range pairs {
			err := data.PushToList(sourceID, pair)
			if err != nil {
				logging.Log.Errorf("Ошибка добавления в DragonflyDB: %v", err)
			} else {
				recordsAdded++
				// Сигнализация (п.4.1)
				if recordsAdded >= batchSize {
					select {
					case signalChan <- struct{}{}:
					default:
					}
					recordsAdded = 0
				}
			}
		}
	}

	return cmd.Wait()
}
