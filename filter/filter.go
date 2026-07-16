package filter

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync/atomic"

	"venera/config"
	"venera/logging"
)

var (
	// Используем atomic.Value для безблокировочного чтения фильтров
	filterData  atomic.Value
	controlData atomic.Value
)

type FilterLists struct {
	WhitelistKeys   map[string]bool
	BlacklistValues map[string]bool
}

type ControlList struct {
	Values map[string]string // Ключ: Значение
}

func init() {
	// Инициализируем пустыми мапами
	filterData.Store(&FilterLists{
		WhitelistKeys:   make(map[string]bool),
		BlacklistValues: make(map[string]bool),
	})
	controlData.Store(&ControlList{
		Values: make(map[string]string),
	})
}

// LoadFilterList загружает списки фильтрации из файла generic.flt.
func LoadFilterList() error {
	newWhitelist := make(map[string]bool)
	newBlacklist := make(map[string]bool)

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

		if strings.HasPrefix(line, "+ ") {
			parts := strings.Split(strings.TrimPrefix(line, "+ "), "|")
			if len(parts) > 0 {
				newWhitelist[strings.TrimSpace(parts[0])] = true
			}
		} else if strings.HasPrefix(line, "- ") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "- "))
			newBlacklist[val] = true
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("ошибка чтения %s: %v", filePath, err)
	}

	filterData.Store(&FilterLists{
		WhitelistKeys:   newWhitelist,
		BlacklistValues: newBlacklist,
	})

	logging.Log.Infof("Загружено правил фильтрации: %d ключей (white), %d значений (black)", len(newWhitelist), len(newBlacklist))
	return nil
}

// LoadControlList загружает список значений на контроле из файла generic.ctr.
func LoadControlList() error {
	newControl := make(map[string]string)

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

		parts := strings.SplitN(line, "|", 2)
		if len(parts) == 2 {
			k := strings.TrimSpace(parts[0])
			v := strings.TrimSpace(parts[1])
			newControl[k] = v
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("ошибка чтения %s: %v", filePath, err)
	}

	controlData.Store(&ControlList{
		Values: newControl,
	})

	logging.Log.Infof("Загружено %d значений на контроле", len(newControl))
	return nil
}

// GetFilterData возвращает текущий снимок фильтров (используется для батч-обработки)
func GetFilterData() (*FilterLists, bool) {
	val := filterData.Load()
	if val == nil {
		return &FilterLists{}, false
	}
	return val.(*FilterLists), true
}

// IsKeyAllowed проверяет, есть ли ключ в белом списке
func IsKeyAllowed(key string) bool {
	fl := filterData.Load().(*FilterLists)
	
	if len(fl.WhitelistKeys) == 0 {
		return true 
	}
	
	return fl.WhitelistKeys[key]
}

// IsValueBlocked проверяет, есть ли значение в черном списке
func IsValueBlocked(value string) bool {
	fl := filterData.Load().(*FilterLists)
	return fl.BlacklistValues[value]
}

// CheckControlValue проверяет, есть ли совпадение в списке контроля
func CheckControlValue(key, value string) bool {
	cl := controlData.Load().(*ControlList)
	
	expectedValue, ok := cl.Values[key]
	if ok && expectedValue == value {
		return true
	}
	return false
}