// Package migrations встраивает SQL схемы в бинарник: в финальный Docker-образ
// каталог migrations не копируется, там лежит только исполняемый файл.
package migrations

import _ "embed"

// CreateURLsTable - схема cut_url и таблица urls. Весь SQL написан через
// IF NOT EXISTS, поэтому его безопасно выполнять при каждом старте сервера.
//
//go:embed create_urls_table.sql
var CreateURLsTable string
