package models

// AlertRule описывает одно правило генерации алерта из файла generic.alr формата cel.
type AlertRule struct {
	ID         string // Уникальный идентификатор правила
	Expression string // CEL выражение (например: "value == 'malicious' && source == 'net1'")
	Msg        string // Текст сообщения для пользователя
	Severity   string // Уровень критичности (info, warning, critical)
}

// AlertEvent описывает событие алерта, которое сгенерировано системой и должно быть показано в UI или логе.
type AlertEvent struct {
	RuleID    string // Ссылка на правило
	Timestamp int64  // Время генерации алерта
	Src       string // Источник данных, вызвавший алерт
	Key       string // Ключ
	Value     string // Подозрительное значение
	Msg       string // Готовое текстовое сообщение
}
