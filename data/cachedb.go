package data

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"venera/config"
	"venera/logging"
)

var (
	CacheDbClient *redis.Client
	ctx           = context.Background()
)

// InitCacheDbConn инициализирует подключение к кэширующей СУБД
func InitCacheDbConn() error {
	cfg := config.GlobalCfg.CacheDb
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	// Повторные попытки подключения с экспоненциальной задержкой
	var err error
	delay := 1 * time.Second

	for i := 0; i < 5; i++ {
		CacheDbClient = redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: cfg.Pass,
			DB:       0, // use default DB
		})

		_, err = CacheDbClient.Ping(ctx).Result()
		if err == nil {
			logging.Log.Infof("Успешное подключение к кэширующей СУБД по адресу %s", addr)
			return nil
		}

		logging.Log.Errorf("Ошибка подключения к кэширующей СУБД (попытка %d/5): %v", i+1, err)
		time.Sleep(delay)
		delay *= 2
	}

	return fmt.Errorf("не удалось подключиться к кэширующей СУБД после 5 попыток: %w", err)
}

// PushBatchToList добавляет пакет записей в структуру List
// Формат: "ключ:значение:время"
func PushBatchToList(srcID string, data []string) error {
	listKey := "list:" + srcID
	err := CacheDbClient.RPush(ctx, listKey, data).Err()
	if err != nil {
		return fmt.Errorf("при добавлении пакета данных в структуру List: %v", err)
	}
	return nil
}

// GetListLen возвращает текущую длину структуры List
func GetListLen(srcID string) (int64, error) {
	listKey := "list:" + srcID
	return CacheDbClient.LLen(ctx, listKey).Result()
}

// PopBatchFromList извлекает пакет записей из структуры List
func PopBatchFromList(srcID string, count int64) ([]string, error) {
	listKey := "list:" + srcID
	
	// Используем транзакцию (pipeline) для получения диапазона и его удаления (LPop count доступно в Redis 6.2+)
	// Для совместимости с DragonflyDB используем LPOP с параметром count
	res, err := CacheDbClient.LPopCount(ctx, listKey, int(count)).Result()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("при извлечении пакета данных из структуры List: %v", err)
	}
	return res, nil
}

// AddToSortedSet добавляет отфильтрованную пару в структуру Sorted Set
// Мы используем srcID как часть ключа, а само значение (ключ:значение) как member, время как score
func AddToSortedSet(srcID string, key, value string, timestamp int64) error {
	setKey := "ss:" + srcID
	member := fmt.Sprintf("%s:%s", key, value)
	
	err := CacheDbClient.ZAdd(ctx, setKey, redis.Z{
		Score:  float64(timestamp),
		Member: member,
	}).Err()
	
	if err != nil {
		return fmt.Errorf("при добавлении данных в структуру Sorted Set: %v", err)
	}
	return nil
}

// GetAndClearSortedSet извлекает все данные из Sorted Set и очищает его
func GetAndClearSortedSet(srcID string) ([]redis.Z, error) {
	setKey := "ss:" + srcID
	
	// Используем ZRange с удалением (если поддерживается, иначе Pipeline)
	// Для простоты используем Pipeline: ZRange -> Del
	pipe := CacheDbClient.Pipeline()
	
	rangeCmd := pipe.ZRangeWithScores(ctx, setKey, 0, -1)
	pipe.Del(ctx, setKey)
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("при извлечении данных из структуры Sorted Set: %v", err)
	}
	
	return rangeCmd.Val(), nil
}

// ParseEntry разбирает строку "ключ:значение:время"
func ParseEntry(entry string) (string, string, string, error) {
	parts := strings.SplitN(entry, ":", 3)
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("неверный формат")
	}
	return parts[0], parts[1], parts[2], nil
}

// CloseCacheDbConn закрывает подключение к кэширующей СУБД
func CloseCacheDbConn() {
	if CacheDbClient != nil {
		_ = CacheDbClient.Close()
	}
}
