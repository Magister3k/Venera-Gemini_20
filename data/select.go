package data

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"

	"venera/logging"
)

var (
	// controlMap хранит список значений на контроле.
	// Формат в мапе: "ключ|значение" (композитный)
	controlMap map[string]struct{}
	controlMu  sync.RWMutex // Защита от состояния гонки
)

// LoadControlList читает файл generic.ctr и заполняет структуру map.
func LoadControlList(filePath string) error {
	controlMu.Lock()
	defer controlMu.Unlock()

	newControlMap := make(map[string]struct{})

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			controlMap = newControlMap
			logging.Log.Warnf("Файл контроля %s не найден. Контроль значений отключен.", filePath)
			return nil
		}
		return fmt.Errorf("ошибка открытия файла контроля: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Хранение списка значений на контроле в форматe csv
	// _ключ_|_значение_
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// Создаем композитный ключ
			compositeKey := key + "|" + value
			newControlMap[compositeKey] = struct{}{}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("ошибка чтения файла контроля: %v", err)
	}

	controlMap = newControlMap
	logging.Log.Infof("Загружен список контроля из %s (записей: %d)", filePath, len(controlMap))
	return nil
}

// IsOnControl проверяет, находится ли пара ключ-значение на контроле (используется для алертов).
func IsOnControl(key, value string) bool {
	controlMu.RLock()
	defer controlMu.RUnlock()

	if len(controlMap) == 0 {
		return false
	}

	compositeKey := key + "|" + value
	_, exists := controlMap[compositeKey]
	return exists
}
