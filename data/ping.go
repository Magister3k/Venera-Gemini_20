package data

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"venera/models"
)

// PingPgDb проверяет подключение к PostgreSQL без инициализации пула глобально
func PingPgDb(cfg *models.PgDbCfg) error {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Pass, cfg.Host, cfg.Port, cfg.Name, cfg.SSLMode)

	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		return err
	}
	defer pool.Close()

	return pool.Ping(context.Background())
}

// PingCacheDb проверяет подключение к DragonflyDB
func PingCacheDb(cfg *models.CacheDbCfg) error {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.Pass,
		DB:       0,
	})
	defer client.Close()

	_, err := client.Ping(context.Background()).Result()
	return err
}
