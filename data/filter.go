package data

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"venera/config"
	"venera/logging"
)

var (
	whitelistKeys    = make(map[string]bool)
	blacklistValues  = make(map[string]bool)
	controlValues    = make(map[string]string) // Ключ: Значение (из generic.ctr)
	filterMu         sync.RWMutex
	controlMu        sync.RWMutex
)

// LoadFilterList загружает списки фильтрации из файла generic.flt (п.1.7).
func LoadFilterList() error {
	filterMu.Lock()
	defer filterMu.Unlock()

	whitelistKeys = make(map[string]bool)
	blacklistValues = make(map[string]bool)

	filePath := config.GlobalConfig.Paths.FilterList
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		logging.Log.Warnf("Файл фильтрации %s не найден", filePath)
		return nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("ошибка открытия %s: %v", filePath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Формат:
		// + ключ|название (белый список ключей)
		// - значение (черный список значений)
		if strings.HasPrefix(line, "+ ") {
			parts := strings.Split(strings.TrimPrefix(line, "+ "), "|")
			if len(parts) > 0 {
				whitelistKeys[strings.TrimSpace(parts[0])] = true
			}
		} else if strings.HasPrefix(line, "- ") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "- "))
			blacklistValues[val] = true
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("ошибка чтения %s: %v", filePath, err)
	}

	logging.Log.Infof("Загружено правил фильтрации: %d ключей (white), %d значений (black)", len(whitelistKeys), len(blacklistValues))
	return nil
}

// LoadControlList загружает список значений на контроле из файла generic.ctr (п.1.8).
func LoadControlList() error {
	controlMu.Lock()
	defer controlMu.Unlock()

	controlValues = make(map[string]string)

	filePath := config.GlobalConfig.Paths.ControlList
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		logging.Log.Warnf("Файл значений на контроле %s не найден", filePath)
		return nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("ошибка открытия %s: %v", filePath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Формат: ключ|значение
		parts := strings.SplitN(line, "|", 2)
		if len(parts) == 2 {
			k := strings.TrimSpace(parts[0])
			v := strings.TrimSpace(parts[1])
			controlValues[k] = v // Можно сохранять как составной ключ, если требуется
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("ошибка чтения %s: %v", filePath, err)
	}

	logging.Log.Infof("Загружено %d значений на контроле", len(controlValues))
	return nil
}

// IsKeyAllowed проверяет, есть ли ключ в белом списке
func IsKeyAllowed(key string) bool {
	filterMu.RLock()
	defer filterMu.RUnlock()
	
	// Если белый список пуст, пропускаем всё (или наоборот блокируем? По ТЗ: "фильтрация ключей согласно белому списку", предполагаем блокировку если не в списке, но для тестов если пусто - разрешаем всё или считаем что список всегда должен быть заполнен. Пусть будет строгая проверка: если список не пуст, ключ должен быть там. Если пуст - пропускаем).
	if len(whitelistKeys) == 0 {
		return true 
	}
	
	return whitelistKeys[key]
}

// IsValueBlocked проверяет, есть ли значение в черном списке
func IsValueBlocked(value string) bool {
	filterMu.RLock()
	defer filterMu.RUnlock()
	return blacklistValues[value]
}

// CheckControlValue проверяет, есть ли совпадение в списке контроля
func CheckControlValue(key, value string) bool {
	controlMu.RLock()
	defer controlMu.RUnlock()
	
	// Простейшая реализация: ищем точное совпадение значения для данного ключа. 
	// (Возможна реализация где ключ не важен, но ТЗ говорит формат: ключ|значение)
	expectedValue, ok := controlValues[key]
	if ok && expectedValue == value {
		return true
	}
	return false
}
