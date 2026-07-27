package models

import	"time"

// DataEntry структура для передачи данных в итоговую БД
type DataEntry struct {
	Source    string
	Key       string
	Value     string
	Timestamp int64
}
