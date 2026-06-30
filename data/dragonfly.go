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

// InitDragonflyDB инициализирует подключение к DragonflyDB
func InitDragonflyDB() error {
	cfg := config.GlobalConfig.DragonflyDB
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	// Повторные попытки подключения с экспоненциальной задержкой
	var err error
	delay := 1 * time.Second

	for i := 0; i < 5; i++ {
		DragonflyClient = redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: cfg.Password,
			DB:       0, // use default DB
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

// PushToList добавляет запись в список (List) для конкретного источника.
// Формат: "ключ:значение:время"
func PushToList(sourceID, data string) error {
	listKey := "list:" + sourceID
	err := DragonflyClient.RPush(ctx, listKey, data).Err()
	if err != nil {
		return fmt.Errorf("ошибка при добавлении в list %s: %v", listKey, err)
	}
	return nil
}

// GetListLength возвращает текущую длину списка
func GetListLength(sourceID string) (int64, error) {
	listKey := "list:" + sourceID
	return DragonflyClient.LLen(ctx, listKey).Result()
}

// PopBatchFromList извлекает до n записей из списка
func PopBatchFromList(sourceID string, count int64) ([]string, error) {
	listKey := "list:" + sourceID
	
	// Используем транзакцию (pipeline) для получения диапазона и его удаления (LPop count доступно в Redis 6.2+)
	// Для совместимости с DragonflyDB используем LPOP с параметром count
	res, err := DragonflyClient.LPopCount(ctx, listKey, int(count)).Result()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("ошибка при извлечении пакета из %s: %v", listKey, err)
	}
	return res, nil
}

// AddToSortedSet добавляет отфильтрованную пару в Sorted Set.
// Мы используем sourceID как часть ключа, а само значение (ключ:значение) как member, время как score
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

// GetAndClearSortedSet извлекает все данные из Sorted Set и очищает его
func GetAndClearSortedSet(sourceID string) ([]redis.Z, error) {
	setKey := "ss:" + sourceID
	
	// Используем ZRange с удалением (если поддерживается, иначе Pipeline)
	// Для простоты используем Pipeline: ZRange -> Del
	pipe := DragonflyClient.Pipeline()
	
	rangeCmd := pipe.ZRangeWithScores(ctx, setKey, 0, -1)
	pipe.Del(ctx, setKey)
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("ошибка при извлечении из sorted set %s: %v", setKey, err)
	}
	
	return rangeCmd.Val(), nil
}

// ParseEntry разбирает строку "ключ:значение:время"
func ParseEntry(entry string) (string, string, string, error) {
	parts := strings.SplitN(entry, ":", 3)
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("неверный формат записи: %s", entry)
	}
	return parts[0], parts[1], parts[2], nil
}

// CloseDragonflyDB закрывает подключение
func CloseDragonflyDB() {
	if DragonflyClient != nil {
		_ = DragonflyClient.Close()
	}
}
