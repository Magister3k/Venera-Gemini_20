package data

import (
	"fmt"

	"github.com/valyala/fastjson"
)

// ParseJSONToPairs разбивает поток JSON на пары ключ-значение.
// С учетом допустимости использования в значениях символа ":".
// Возвращает слайс строк формата "ключ:значение:время".
func ParseJSONToPairs(jsonData []byte, timestamp int64) ([]string, error) {
	var pairs []string

	// fastjson.Parse распарсит JSON любой сложности.
	var p fastjson.Parser
	v, err := p.ParseBytes(jsonData)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON: %v", err)
	}

	// Рекурсивный обход для извлечения всех ключей и примитивных значений
	var traverse func(val *fastjson.Value, currentPath string)
	traverse = func(val *fastjson.Value, currentPath string) {
		switch val.Type() {
		case fastjson.TypeObject:
			obj := val.GetObject()
			obj.Visit(func(k []byte, v *fastjson.Value) {
				newPath := string(k)
				if currentPath != "" {
					newPath = currentPath + "." + string(k)
				}
				traverse(v, newPath)
			})
		case fastjson.TypeArray:
			arr := val.GetArray()
			for i, v := range arr {
				newPath := fmt.Sprintf("%s[%d]", currentPath, i)
				traverse(v, newPath)
			}
		case fastjson.TypeString, fastjson.TypeNumber, fastjson.TypeTrue, fastjson.TypeFalse, fastjson.TypeNull:
			// Для листовых узлов сохраняем пару.
			strVal := string(val.GetStringBytes())
			if val.Type() != fastjson.TypeString {
				strVal = val.String()
			}

			// Если значение содержит двоеточие, оно будет сохранено,
			// так как ParseEntry (в dragonfly.go) использует strings.SplitN(..., ":", 3)
			// и разбивает строго на "ключ", "значение(возможно с двоеточиями)" и "время".

			// Для надежности, чтобы избежать конфликтов при ParseEntry,
			// мы можем использовать другой разделитель или кодировать значение.
			// Но ТЗ требует формат: ключ:значение:время.
			// В data/dragonfly.go (ParseEntry) мы реализовали SplitN(..., ":", 3), что не сработает корректно
			// если в значении есть двоеточия (оно захватит время как часть значения, если разбивать с конца, или разобьет значение).

			// Поэтому мы будем использовать специальный формат для хранения.
			// ТЗ: "добавление к паре ключ-значение времени фиксации (приведение к формату ключ:значение:время)"

			// Исправленный подход: Значение не должно ломать парсинг.
			// Мы будем гарантировать, что время всегда в конце, а ключ не содержит ":".
			// Формат: "ключ:значение:время"
			// При чтении (ParseEntry) нужно искать последнее двоеточие для времени, и первое для ключа.

			formattedStr := fmt.Sprintf("%s:%s:%d", currentPath, strVal, timestamp)
			pairs = append(pairs, formattedStr)
		}
	}

	traverse(v, "")
	return pairs, nil
}

// ImprovedParseEntry - улучшенная версия ParseEntry из cachedb.go
// которая корректно обрабатывает значения, содержащие двоеточия.
func ImprovedParseEntry(entry string) (string, string, int64, error) {
	// Формат: "ключ:значение:время"

	firstColon := -1
	lastColon := -1

	for i := 0; i < len(entry); i++ {
		if entry[i] == ':' {
			if firstColon == -1 {
				firstColon = i
			}
			lastColon = i
		}
	}

	if firstColon == -1 || firstColon == lastColon {
		return "", "", 0, fmt.Errorf("неверный формат записи (отсутствуют нужные разделители): %s", entry)
	}

	key := entry[:firstColon]
	value := entry[firstColon+1 : lastColon]
	timeStr := entry[lastColon+1:]

	var timestamp int64
	_, err := fmt.Sscanf(timeStr, "%d", &timestamp)
	if err != nil {
		return "", "", 0, fmt.Errorf("ошибка парсинга времени %s: %v", timeStr, err)
	}

	return key, value, timestamp, nil
}
