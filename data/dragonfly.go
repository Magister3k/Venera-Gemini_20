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
	DragonflyClient *redis.Client
	ctx             = context.Background()
)

func InitDragonflyDB() error {
	cfg := config.GlobalConfig.DragonflyDB
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	var err error
	delay := 1 * time.Second

	for i := 0; i < 5; i++ {
		DragonflyClient = redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: cfg.Password,
			DB:       0,
		})

		_, err = DragonflyClient.Ping(ctx).Result()
		if err == nil {
			logging.Log.Infof("Успешное подключение к DragonflyDB по адресу %s", addr)
			return nil
		}

		logging.Log.Warnf("Ошибка подключения к DragonflyDB (попытка %d/5): %v", i+1, err)
		time.Sleep(delay)
		delay *= 2
	}

	return fmt.Errorf("не удалось подключиться к DragonflyDB после 5 попыток: %w", err)
}

func PushToList(sourceID, data string) error {
	listKey := "list:" + sourceID
	err := DragonflyClient.RPush(ctx, listKey, data).Err()
	if err != nil {
		return fmt.Errorf("ошибка при добавлении в list %s: %v", listKey, err)
	}
	return nil
}

func GetListLength(sourceID string) (int64, error) {
	listKey := "list:" + sourceID
	return DragonflyClient.LLen(ctx, listKey).Result()
}

func PopBatchFromList(sourceID string, count int64) ([]string, error) {
	listKey := "list:" + sourceID
	
	if count == -1 {
		// Получить все элементы и удалить
		pipe := DragonflyClient.Pipeline()
		rangeCmd := pipe.LRange(ctx, listKey, 0, -1)
		pipe.Del(ctx, listKey)
		_, err := pipe.Exec(ctx)
		if err != nil {
			return nil, err
		}
		return rangeCmd.Val(), nil
	}
	
	res, err := DragonflyClient.LPopCount(ctx, listKey, int(count)).Result()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("ошибка при извлечении пакета из %s: %v", listKey, err)
	}
	return res, nil
}

func AddToSortedSet(sourceID string, key, value string, timestamp int64) error {
	setKey := "ss:" + sourceID
	member := fmt.Sprintf("%s:%s", key, value)
	
	err := DragonflyClient.ZAdd(ctx, setKey, redis.Z{
		Score:  float64(timestamp),
		Member: member,
	}).Err()
	
	if err != nil {
		return fmt.Errorf("ошибка при добавлении в sorted set %s: %v", setKey, err)
	}
	return nil
}

// CleanupSortedSet удаляет старые записи из Sorted Set (например, старше 24 часов)
func CleanupSortedSet(sourceID string, olderThan int64) error {
	setKey := "ss:" + sourceID
	err := DragonflyClient.ZRemRangeByScore(ctx, setKey, "-inf", fmt.Sprintf("%d", olderThan)).Err()
	return err
}

func ParseEntry(entry string) (string, string, string, error) {
	parts := strings.SplitN(entry, ":", 3)
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("неверный формат записи: %s", entry)
	}
	return parts[0], parts[1], parts[2], nil
}

func CloseDragonflyDB() {
	if DragonflyClient != nil {
		_ = DragonflyClient.Close()
	}
}