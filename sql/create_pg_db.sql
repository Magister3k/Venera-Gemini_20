package sql

const CreatePGDBSql = `
-- Создание базы данных (выполняется отдельно, так как CREATE DATABASE нельзя выполнить в блоке)
-- CREATE DATABASE Venera;

-- Создание таблицы для данных (п.3)
CREATE TABLE IF NOT EXISTS venera_data (
    id SERIAL PRIMARY KEY,
    source VARCHAR(255) NOT NULL,
    key VARCHAR(255) NOT NULL,
    value TEXT NOT NULL,
    date_first TIMESTAMP NOT NULL,
    date_last TIMESTAMP NOT NULL,
    UNIQUE(source, key, value)
);

-- Индексы для быстрого поиска
CREATE INDEX IF NOT EXISTS idx_venera_data_source ON venera_data(source);
CREATE INDEX IF NOT EXISTS idx_venera_data_key ON venera_data(key);
CREATE INDEX IF NOT EXISTS idx_venera_data_date_last ON venera_data(date_last);
`
