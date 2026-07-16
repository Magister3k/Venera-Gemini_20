package processes

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"time"

	"golang.org/x/sys/windows"
	"venera/config"
	"venera/data"
	"venera/logging"
)

// RunTsharkNetwork запускает Tshark для захвата с сетевого интерфейса
func RunTsharkNetwork(ctx context.Context, ip string, port int, sourceID string, trigger func()) error {
	exe := config.GlobalConfig.Paths.TsharkExe
	filter := fmt.Sprintf("host %s and udp port %d", ip, port)
	// -T ek выдает каждый JSON объект на новой строке (NDJSON)
	args := []string{"-i", "any", "-f", filter, "-T", "ek"}

	return runTsharkCommand(ctx, exe, args, sourceID, trigger)
}

// RunTsharkFile запускает Tshark для чтения из файла
func RunTsharkFile(ctx context.Context, filePath, folderPath, sourceID string, trigger func()) error {
	exe := config.GlobalConfig.Paths.TsharkExe
	var args []string

	if filePath != "" {
		args = []string{"-r", filePath, "-T", "ek"}
	} else if folderPath != "" {
		return fmt.Errorf("обработка папки не реализована в заглушке, требует итерации")
	} else {
		return fmt.Errorf("не указан путь к файлу или папке")
	}

	return runTsharkCommand(ctx, exe, args, sourceID, trigger)
}

func runTsharkCommand(ctx context.Context, exe string, args []string, sourceID string, trigger func()) error {
	cmd := exec.CommandContext(ctx, exe, args...)

	// Настройка для Windows, чтобы Tshark убивался при закрытии основного процесса.
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

	// Канал для синхронизации завершения горутины чтения stderr
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

	batchSize := int64(config.GlobalConfig.DragonflyDB.BatchSize)
	var recordsAdded int64

	scanner := bufio.NewScanner(stdout)
	// Увеличиваем буфер, так как JSON может быть большим
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		
		// В формате -T ek выводятся строки индекса и данные.
		// Строки индекса {"index":{...}} нам не нужны, пропускаем.
		if len(line) > 0 && line[0] == '{' && string(line[1:8]) == "\"index\"" {
			continue
		}

		ts := time.Now().UnixMilli()

		pairs, err := data.ParseJSONToPairs(line, ts)
		if err != nil || len(pairs) == 0 {
			continue
		}

		// Используем батч добавление в Redis для минимизации round-trip задержек
		err = data.PushBatchToList(sourceID, pairs)
		if err != nil {
			logging.Log.Errorf("Ошибка добавления батча в DragonflyDB: %v", err)
		} else {
			recordsAdded += int64(len(pairs))
			if recordsAdded >= batchSize {
				trigger()
				recordsAdded = 0
			}
		}
	}

	if err := scanner.Err(); err != nil {
		logging.Log.Errorf("Tshark stdout error: %v", err)
	}

	// Ожидаем завершения чтения stderr, чтобы предотвратить race condition и ошибки "file already closed"
	<-stderrDone

	return cmd.Wait()
}