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
	// filterMap хранит двухуровневый список фильтрации.
	// Первый уровень: ключ (белый список "+").
	// Второй уровень: список значений (черный список "-").
	// Для быстрого поиска значений используется map[string]struct{}.
	filterMap map[string]map[string]struct{}
	filterMu  sync.RWMutex // Защита от состояния гонки (Race Condition), выявленного в п.27
)

// LoadFilters читает файл generic.flt и заполняет двухуровневую структуру map (п.2.5 ТЗ).
func LoadFilters(filePath string) error {
	filterMu.Lock()
	defer filterMu.Unlock()

	// Инициализация новой пустой мапы
	newFilterMap := make(map[string]map[string]struct{})

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// ТЗ п.2.5: "Проверка наличия файла... в случае наличия - загрузка".
			// Отсутствие файла не является фатальной ошибкой, просто список будет пуст.
			filterMap = newFilterMap
			logging.Log.Warnf("Файл фильтров %s не найден. Фильтрация отключена.", filePath)
			return nil
		}
		return fmt.Errorf("ошибка открытия файла фильтров: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var currentKey string

	// ТЗ п.1.7:
	// "+" _ключ_|_название_
	// "-" _значение_
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "+ ") {
			// Парсим ключ белого списка
			parts := strings.SplitN(line[2:], "|", 2)
			if len(parts) > 0 {
				currentKey = strings.TrimSpace(parts[0])
				if _, exists := newFilterMap[currentKey]; !exists {
					newFilterMap[currentKey] = make(map[string]struct{})
				}
			}
		} else if strings.HasPrefix(line, "- ") {
			// Парсим значение черного списка для текущего ключа
			if currentKey != "" {
				value := strings.TrimSpace(line[2:])
				newFilterMap[currentKey][value] = struct{}{}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("ошибка чтения файла фильтров: %v", err)
	}

	filterMap = newFilterMap
	logging.Log.Infof("Загружен список фильтрации из %s (ключей: %d)", filePath, len(filterMap))
	return nil
}

// IsAllowed проверяет ключ и значение на соответствие правилам фильтрации (п.5.5.3 ТЗ).
// Возвращает true, если пара должна быть обработана, и false, если она отбрасывается.
func IsAllowed(key, value string) bool {
	filterMu.RLock()
	defer filterMu.RUnlock()

	// Если список фильтров пуст, разрешаем все (или можно изменить логику на запрет всего)
	if len(filterMap) == 0 {
		return true
	}

	// 1. Фильтрация ключей согласно белому списку (если ключ есть в мапе)
	blackListValues, keyExists := filterMap[key]
	if !keyExists {
		return false // Ключ не прошел белый список
	}

	// 2. Фильтрация значений согласно черному списку
	// Если список значений для данного ключа пуст - разрешены все значения.
	if len(blackListValues) == 0 {
		return true
	}

	// Если значение найдено в черном списке, отбрасываем
	if _, isBlacklisted := blackListValues[value]; isBlacklisted {
		return false
	}

	// Иначе разрешаем
	return true
}

// GetFilterKeys возвращает список всех ключей в белом списке.
// Может понадобиться для GUI или диагностики.
func GetFilterKeys() []string {
	filterMu.RLock()
	defer filterMu.RUnlock()

	keys := make([]string, 0, len(filterMap))
	for k := range filterMap {
		keys = append(keys, k)
	}
	return keys
}
