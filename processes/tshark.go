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

// RunTsharkNet запускает Tshark для захвата с сетевого интерфейса
func RunTsharkNet(ctx context.Context, ip string, port int, srcID string, trigger func()) error {
	exe := config.GlobalCfg.Paths.Tshark
	filter := fmt.Sprintf("host %s and udp port %d", ip, port)
	// -T ek выдает каждый JSON объект на новой строке (NDJSON)
	args := []string{"-i", "any", "-f", filter, "-T", "ek"}

	return runTsharkCmd(ctx, exe, args, srcID, trigger)
}

// RunTsharkFileOrDir запускает обработку отдельного файла или папки с файлами
func RunTsharkFileOrDir(ctx context.Context, p models.ProcCfg, trigger func()) error {
	exe := config.GlobalCfg.Paths.Tshark

	if p.Type == models.SrcFile && p.FilePath != "" {
		// Обработка одного файла
		args := []string{"-r", p.FilePath, "-T", "ek"}
		return runTsharkCmd(ctx, exe, args, p.ID, trigger)
	}

	if p.Type == models.SrcDir && p.DirPath != "" {
		// Обработка файлов в папке
		return procDir(ctx, exe, p, trigger)
	}

	return fmt.Errorf("не указан путь к файлу или папке для источника: %s", p.ID)
}

// procDir обрабатывает папку с pcap файлами с учетом параметров подпапок и мониторинга
func procDir(ctx context.Context, exe string, p models.ProcCfg, trigger func()) error {
	doneFiles := make(map[string]bool)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var inFiles []string

		// Функция обхода файлов
		walkFunc := func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				if path != p.DirPath && !p.ScanSubdirs {
					return filepath.SkipDir // Пропускаем подпапки, если выключено
				}
				return nil
			}

			// Проверяем расширения (для pcap/pcapng) и то, что файл еще не обработан
			ext := filepath.Ext(path)
			if (ext == ".pcap" || ext == ".pcapng") && !doneFiles[path] {
				inFiles = append(inFiles, path)
			}
			return nil
		}

		err := filepath.Walk(p.DirPath, walkFunc)
		if err != nil {
			logging.Log.Errorf("Ошибка сканирования папки %s: %v", p.DirPath, err)
		}

		// Обрабатываем найденные файлы последовательно
		for _, file := range inFiles {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			logging.Log.Infof("Обработка файла из папки: %s", file)
			args := []string{"-r", file, "-T", "ek"}
			err = runTsharkCmd(ctx, exe, args, p.ID, trigger)
			if err != nil && err != context.Canceled {
				logging.Log.Warnf("Ошибка обработки файла %s: %v", file, err)
			}

			doneFiles[file] = true // Отмечаем как обработанный
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

// runTsharkCmd запускает команду Tshark и читает её STDOUT
func runTsharkCmd(ctx context.Context, exe string, args []string, srcID string, trigger func()) error {
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
			logging.Log.Warnf("Tshark [%s] stderr: %s", srcID, errText)
		}
	}()

	batchSize := int64(config.GlobalCfg.CacheDb.BatchSize)
	var recsAdded int64

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

		// Добавление пачки данных в кэширующую СУБД
		err = data.PushBatchToList(srcID, pairs)
		if err != nil {
			logging.Log.Errorf("Ошибка добавления данных в кэширующую СУБД: %v", err)
		} else {
			recsAdded += int64(len(pairs))
			if recsAdded >= batchSize {
				trigger() // Вызов канала порогового значения
				recsAdded = 0
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

