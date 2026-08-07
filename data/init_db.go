package data

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"venera/assets"
	"venera/logging"
	"venera/models"
)

// InitPgDb выполняет команду инициализации итоговой базы в СУБД PostgreSQL.
func InitPgDb(cfg *models.PgDbCfg) error {
	ctx := context.Background()

	// Подключаемся к СУБД PostgreSQL, чтобы проверить существование базы
	sysConnStr := fmt.Sprintf("postgres://%s:%s@%s:%d/postgres?sslmode=%s",
		cfg.User, cfg.Pass, cfg.Host, cfg.Port, cfg.SSLMode)

	sysPool, err := pgxpool.New(ctx, sysConnStr)
	if err != nil {
		return fmt.Errorf("ошибка подключения к СУБД PostgreSQL: %v", err)
	}
	defer sysPool.Close()

	// Проверяем наличие итоговой базы данных
	var exists bool
	queryCheck := `SELECT EXISTS(SELECT datname FROM pg_catalog.pg_database WHERE datname = $1);`
	err = sysPool.QueryRow(ctx, queryCheck, cfg.Name).Scan(&exists)
	if err != nil {
		return fmt.Errorf("ошибка проверки существования базы: %v", err)
	}

	if !exists {
		logging.Log.Infof("База данных '%s' не найдена. Начинаем создание...", cfg.Name)
		// Команду CREATE DATABASE нельзя выполнить как параметризованный запрос,
		// поэтому мы подставляем имя базы данных напрямую, предварительно проверив на безопасность (оно из конфига).
		// Экранируем двойными кавычками на случай спецсимволов.
		createDbQuery := fmt.Sprintf(`CREATE DATABASE "%s";`, strings.ReplaceAll(cfg.Name, `"`, `""`))

		_, err = sysPool.Exec(ctx, createDbQuery)
		if err != nil {
			return fmt.Errorf("ошибка создания базы данных: %v", err)
		}
		logging.Log.Infof("База данных '%s' успешно создана.", cfg.Name)
	} else {
		logging.Log.Infof("База данных '%s' уже существует. Применение схемы...", cfg.Name)
	}

	// Закрываем подключение к системной базе данных и подключаемся к созданной (или существующей) базе данных
	targetConnStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Pass, cfg.Host, cfg.Port, cfg.Name, cfg.SSLMode)

	targetPool, err := pgxpool.New(ctx, targetConnStr)
	if err != nil {
		return fmt.Errorf("ошибка подключения к итоговой базе '%s': %v", cfg.Name, err)
	}
	defer targetPool.Close()

	// Выполняем SQL-скрипт создания базы данных
	_, err = targetPool.Exec(ctx, assets.SchemaSQL)
	if err != nil {
		return fmt.Errorf("ошибка применения шаблона SQL-скрипта: %v", err)
	}

	logging.Log.Infof("Таблицы и индексы базы данных '%s' успешно инициализированы.", cfg.Name)
	return nil
}
