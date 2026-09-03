package storage

import (
	"database/sql"
	_ "embed"
	"fmt"
	_ "modernc.org/sqlite"
)

//go:embed migrations/001_init.sql
var initMigration string

func NewSQLite(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("не удалось подключиться к базе данных: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("не удалось установить соединение с базой данных: %w", err)
	}

	return db, nil
}

func Migrate(db *sql.DB) error {
	_, err := db.Exec(initMigration)
	if err != nil {
		return fmt.Errorf("не удалось применить миграцию: %w", err)
	}
	return nil
}
