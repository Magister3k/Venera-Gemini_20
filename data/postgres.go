package data

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"venera/config"
	"venera/logging"
	"venera/models"
)

var (
	PgPool *pgxpool.Pool
)

// InitPgConn инициализирует пул подключений к базе в СУБД PostgreSQL
func InitPgConn() error {
	cfg := config.GlobalConfig.PostgreSQL
	connString := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database, cfg.SSLMode)

	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return fmt.Errorf("ошибка парсинга строки подключения к базе в СУБД PostgreSQL: %v", err)
	}

	delay := 1 * time.Second
	for i := 0; i < 5; i++ {
		PgPool, err = pgxpool.NewWithConfig(context.Background(), poolConfig)
		if err == nil {
			err = PgPool.Ping(context.Background())
			if err == nil {
				logging.Log.Infof("Успешное подключение к базе в СУБД PostgreSQL по адресу %s:%d", cfg.Host, cfg.Port)
				return nil
			}
		}

		logging.Log.Warnf("Ошибка подключения к базе в СУБД PostgreSQL (попытка %d/5): %v", i+1, err)
		if PgPool != nil {
			PgPool.Close()
		}
		time.Sleep(delay)
		delay *= 2
	}

	return fmt.Errorf("не удалось подключиться к базе в СУБД PostgreSQL после 5 попыток: %w", err)
}

// InsertBatch выполняет пакетную вставку или обновление данных в PostgreSQL
func InsertBatch(sourceID string, entries []DataEntry) error {
	if len(entries) == 0 {
		return nil
	}

	ctx := context.Background()

	// В PostgreSQL 15+ можно использовать MERGE или делать UPSERT через временную таблицу для скорости.
	// Для совместимости используем Batch с ON CONFLICT.
	batch := &pgx.Batch{}

	// SQL-запрос (UPSERT).
	// Используем to_timestamp($4::double precision) для сохранения миллисекунд
	query := `
		INSERT INTO venera_data (source, key, value, date_first, date_last)
		VALUES ($1, $2, $3, to_timestamp($4::double precision), to_timestamp($4::double precision))
		ON CONFLICT (source, key, value) 
		DO UPDATE SET date_last = GREATEST(venera_data.date_last, to_timestamp($4::double precision));
	`

	for _, entry := range entries {
		tsSeconds := float64(entry.Timestamp) / 1000.0
		batch.Queue(query, entry.Source, entry.Key, entry.Value, tsSeconds)
	}

	br := PgPool.SendBatch(ctx, batch)
	err := br.Close()
	if err != nil {
		return fmt.Errorf("ошибка выполнения пакета запросов: %v", err)
	}

	return nil
}

// ClosePgConn закрывает пул подключений к базе в СУБД PostgreSQL
func ClosePgConn() {
	if PgPool != nil {
		PgPool.Close()
	}
}