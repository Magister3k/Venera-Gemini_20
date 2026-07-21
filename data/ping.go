package data

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"venera/models"
)

// PingPostgres проверяет подключение к PostgreSQL без инициализации пула глобально
func PingPostgres(cfg *models.PostgreSQLConfig) error {
	connString := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database, cfg.SSLMode)

	pool, err := pgxpool.New(context.Background(), connString)
	if err != nil {
		return err
	}
	defer pool.Close()

	return pool.Ping(context.Background())
}

// PingDragonfly проверяет подключение к DragonflyDB
func PingDragonfly(cfg *models.DragonflyDBConfig) error {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.Password,
		DB:       0,
	})
	defer client.Close()

	_, err := client.Ping(context.Background()).Result()
	return err
}
