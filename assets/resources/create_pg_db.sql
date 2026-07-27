-- Шаблон SQL-скрипта для создания таблиц базы данных "Venera" (п.20 ТЗ)
-- Этот скрипт выполняется после создания самой БД "Venera".

-- ВНИМАНИЕ: Если вы выполняете скрипт вручную, сначала выполните:
-- CREATE DATABASE "Venera";
-- \c "Venera";

-- Создание таблицы для данных в соответствии со структурой из п.3 ТЗ:
CREATE TABLE IF NOT EXISTS venera_data (
    -- Внутренний суррогатный ключ (для удобства ORM и администрирования, хоть и не указан в ТЗ)
    id SERIAL PRIMARY KEY,
    
    -- Название источника (п.3 ТЗ)
    source VARCHAR(255) NOT NULL,
    
    -- Ключ (п.3 ТЗ)
    key VARCHAR(255) NOT NULL,
    
    -- Значение (п.3 ТЗ)
    value TEXT NOT NULL,
    
    -- Дата первого появления значения (п.3 ТЗ)
    date_first TIMESTAMP NOT NULL,
    
    -- Дата последнего появления значения (п.3 ТЗ)
    date_last TIMESTAMP NOT NULL,
    
    -- Уникальное ограничение для UPSERT (ON CONFLICT) из data/postgres.go
    CONSTRAINT unique_source_key_value UNIQUE(source, key, value)
);

-- Создание индексов для обеспечения высокой производительности выборок
-- в веб-интерфейсе (п.10.6 ТЗ) и ускорения работы ON CONFLICT (п.5.5.5 ТЗ).
CREATE INDEX IF NOT EXISTS idx_venera_data_source ON venera_data(source);
CREATE INDEX IF NOT EXISTS idx_venera_data_key ON venera_data(key);
CREATE INDEX IF NOT EXISTS idx_venera_data_date_last ON venera_data(date_last);
