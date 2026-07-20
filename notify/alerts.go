package notify

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"venera/logging"
	"venera/models"
)

var (
	alertRules []compiledRule	// скомпилированные правила алертов
	alertsMu   sync.RWMutex
	celEnv     *cel.Env		// окружение CEL для компиляции выражений
)

// compiledRule - исходное и скомпилированное правило алерта в формате CEL
type compiledRule struct {
	Rule    models.AlertRule	// исходное правило
	Program cel.Program		// скомпилированное правило
}

func init() {
	// Инициализация окружения CEL с определением переменных,
	// которые могут использоваться в файле generic.alr
	var err error
	celEnv, err = cel.NewEnv(
		cel.Declarations(
			decls.NewVar("source", decls.String),
			decls.NewVar("key", decls.String),
			decls.NewVar("value", decls.String),
		),
	)
	if err != nil {
		panic(fmt.Sprintf("Ошибка инициализации CEL окружения: %v", err))
	}
}

// LoadAlerts загружает правила алертов в формате CEL из файла CSV (generic.alr)
// Ожидаемый формат файла (строчный, разделитель "|"):
// Message | Severity | CEL_Expression
// Пример: Обнаружен админ | warning | value == 'admin' && key == 'user'
func LoadAlerts(filePath string) error {
	alertsMu.Lock()
	defer alertsMu.Unlock()

	newRules := make([]compiledRule, 0)

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			alertRules = newRules
			logging.Log.Warnf("Файл алертов %s не найден. Алерты отключены", filePath)
			return nil
		}
		return fmt.Errorf("ошибка открытия файла алертов: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Пропуск пустых строк и комментариев
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "|", 3)
		if len(parts) != 3 {
			logging.Log.Warnf("Ошибка парсинга алерта в строке %d: ожидается формат Message|Severity|Expression", lineNum)
			continue
		}

		msg := strings.TrimSpace(parts[0])
		severity := strings.TrimSpace(parts[1])
		expr := strings.TrimSpace(parts[2])

		// Компиляция CEL выражения
		ast, issues := celEnv.Compile(expr)
		if issues != nil && issues.Err() != nil {
			logging.Log.Warnf("Ошибка компиляции CEL алерта в строке %d: %v", lineNum, issues.Err())
			continue
		}

		prg, err := celEnv.Program(ast)
		if err != nil {
			logging.Log.Warnf("Ошибка создания CEL программы в строке %d: %v", lineNum, err)
			continue
		}

		rule := models.AlertRule{
			ID:         fmt.Sprintf("rule_%d", lineNum),
			Message:    msg,
			Severity:   severity,
			Expression: expr,
		}

		newRules = append(newRules, compiledRule{
			Rule:    rule,
			Program: prg,
		})
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("ошибка чтения файла алертов: %v", err)
	}

	alertRules = newRules
	logging.Log.Infof("Загружены правила алертов: %d правил из %s", len(alertRules), filePath)
	return nil
}

// CheckAlerts проверяет конкретную пару данных на соответствие загруженным CEL правилам
func CheckAlerts(source, key, value string) {
	alertsMu.RLock()
	defer alertsMu.RUnlock()

	if len(alertRules) == 0 {
		return
	}

	// Формируем карту переменных для CEL
	vars := map[string]interface{}{
		"source": source,
		"key":    key,
		"value":  value,
	}

	for _, cr := range alertRules {
		out, _, err := cr.Program.Eval(vars)
		if err != nil {
			logging.Log.Errorf("Ошибка выполнения CEL алерта %s: %v", cr.Rule.ID, err)
			continue
		}

		// Если результат выражения == true
		if out.Value() == true {
			event := models.AlertEvent{
				RuleID:    cr.Rule.ID,
				Timestamp: time.Now().UnixMilli(),
				Source:    source,
				Key:       key,
				Value:     value,
				Message:   cr.Rule.Message,
			}

			// Отправка алерта
			ProcessAlert(event, cr.Rule.Severity)
		}
	}
}
