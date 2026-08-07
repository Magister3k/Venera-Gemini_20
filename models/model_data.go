package models

// DataEntry представляет одну разобранную запись (ключ-значение-время), готовую для вставки в итоговую базу.
type DataEntry struct {
	Source    string // Название источника
	Key       string // Ключ
	Value     string // Значение
	Timestamp int64  // Unix timestamp (мс) (используется для date_first и date_last через UPSERT)
}
