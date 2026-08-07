package assets
// Внедрение файлов в бинарник приложения

import _ "embed"

// Внедряем иконку системного трея
//go:embed resources/image.ico
var IconBytes []byte


// Внедряем SQL-скрипт создания базы в СУБД PostgreSQL
//
//go:embed resources/create_db.sql
var SchemaSQL string
