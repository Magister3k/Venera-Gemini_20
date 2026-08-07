package processes

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows"
	"venera/config"
	"venera/data"
	"venera/logging"
	"venera/models"
)

// RunTsharkNetwork запускает Tshark для захвата с сетевого интерфейса
func RunTsharkNetwork(ctx context.Context, ip string, port int, sourceID string, trigger func()) error {
	exe := config.GlobalCfg.Paths.Tshark
	filter := fmt.Sprintf("host %s and udp port %d", ip, port)
	// -T ek выдает каждый JSON объект на новой строке (NDJSON)
	args := []string{"-i", "any", "-f", filter, "-T", "ek"}

	return runTsharkCommand(ctx, exe, args, sourceID, trigger)
}

// RunTsharkFileOrFolder запускает обработку отдельного файла или папки с файлами
func RunTsharkFileOrFolder(ctx context.Context, p models.ProcessConfig, trigger func()) error {
	exe := config.GlobalCfg.Paths.Tshark

	if p.Type == models.SourceFile && p.FilePath != "" {
		// Обработка одного файла
		args := []string{"-r", p.FilePath, "-T", "ek"}
		return runTsharkCommand(ctx, exe, args, p.ID, trigger)
	}

	if p.Type == models.SourceFolder && p.FolderPath != "" {
		// Обработка файлов в папке
		return processFolder(ctx, exe, p, trigger)
	}

	return fmt.Errorf("не указан путь к файлу или папке для источника: %s", p.ID)
}

// processFolder обрабатывает папку с pcap файлами с учетом параметров подпапок и мониторинга
func processFolder(ctx context.Context, exe string, p models.ProcessConfig, trigger func()) error {
	processedFiles := make(map[string]bool)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var filesToProcess []string

		// Функция обхода файлов
		walkFunc := func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				if path != p.FolderPath && !p.ScanSubfolders {
					return filepath.SkipDir // Пропускаем подпапки, если выключено
				}
				return nil
			}

			// Проверяем расширения (для pcap/pcapng) и то, что файл еще не обработан
			ext := filepath.Ext(path)
			if (ext == ".pcap" || ext == ".pcapng") && !processedFiles[path] {
				filesToProcess = append(filesToProcess, path)
			}
			return nil
		}

		err := filepath.Walk(p.FolderPath, walkFunc)
		if err != nil {
			logging.Log.Errorf("Ошибка сканирования папки %s: %v", p.FolderPath, err)
		}

		// Обрабатываем найденные файлы последовательно
		for _, file := range filesToProcess {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			logging.Log.Infof("Обработка файла из папки: %s", file)
			args := []string{"-r", file, "-T", "ek"}
			err = runTsharkCommand(ctx, exe, args, p.ID, trigger)
			if err != nil && err != context.Canceled {
				logging.Log.Warnf("Ошибка обработки файла %s: %v", file, err)
			}

			processedFiles[file] = true // Отмечаем как обработанный
		}

		// Если мониторинг новых файлов выключен, выходим после одного прохода
		if !p.MonitorNewFiles {
			break
		}

		// Ждем перед следующим сканированием папки
		time.Sleep(5 * time.Second)
	}

	return nil
}

// runTsharkCommand запускает команду Tshark и читает её STDOUT
func runTsharkCommand(ctx context.Context, exe string, args []string, sourceID string, trigger func()) error {
	cmd := exec.CommandContext(ctx, exe, args...)

	// Настройка для Windows: Job Objects или группы процессов (защита от зомби-процессов Tshark)
	cmd.SysProcAttr = &windows.SysProcAttr{
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP,
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("ошибка получения stdout Tshark: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("ошибка получения stderr Tshark: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("ошибка запуска Tshark: %v", err)
	}

	stderrDone := make(chan struct{})

	// Чтение ошибок Tshark в отдельной горутине
	go func() {
		defer close(stderrDone)
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			errText := scanner.Text()
			logging.Log.Warnf("Tshark [%s] stderr: %s", sourceID, errText)
		}
	}()

	batchSize := int64(config.GlobalCfg.DragonflyDB.BatchSize)
	var recordsAdded int64

	scanner := bufio.NewScanner(stdout)
	// Увеличиваем буфер, так как JSON строки могут быть длинными
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 5*1024*1024) // До 5 МБ на строку

	for scanner.Scan() {
		line := scanner.Bytes()

		// -T ek выводит строки индекса перед данными, пропускаем их
		if len(line) > 0 && line[0] == '{' && string(line[1:8]) == "\"index\"" {
			continue
		}

		ts := time.Now().UnixMilli()

		pairs, err := data.ParseJSONToPairs(line, ts)
		if err != nil || len(pairs) == 0 {
			continue
		}

		// Пачки в кэширующей СУБД
		err = data.PushBatchToList(sourceID, pairs)
		if err != nil {
			logging.Log.Errorf("Ошибка добавления пачки в кэширующую СУБД: %v", err)
		} else {
			recordsAdded += int64(len(pairs))
			if recordsAdded >= batchSize {
				trigger() // Вызов канала порогового значения
				recordsAdded = 0
			}
		}
	}

	if err := scanner.Err(); err != nil {
		// Ошибка чтения (может быть context canceled или EOF)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		logging.Log.Errorf("Ошибка вывода Tshark: %v", err)
	}

	// Ожидаем завершения чтения stderr, предотвращая panic/race condition
	<-stderrDone

	return cmd.Wait()
}
