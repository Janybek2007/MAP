package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func OpenAndMigrate(databasePath string) (*sql.DB, error) {
	cleanPath := filepath.Clean(databasePath)
	if err := os.MkdirAll(filepath.Dir(cleanPath), 0o755); err != nil {
		return nil, fmt.Errorf("не удалось создать директорию sqlite: %w", err)
	}

	database, err := sql.Open("sqlite", cleanPath)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть sqlite: %w", err)
	}

	if err := database.Ping(); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("не удалось подключиться к sqlite: %w", err)
	}

	if _, err := database.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("не удалось включить внешние ключи sqlite: %w", err)
	}

	if err := applyInitialSchema(database); err != nil {
		_ = database.Close()
		return nil, err
	}

	return database, nil
}

func OpenExistingAndMigrate(databasePath string) (*sql.DB, error) {
	cleanPath := filepath.Clean(databasePath)
	fileInfo, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("sqlite не найден по пути %s; сначала выполни import-sqlite", cleanPath)
		}
		return nil, fmt.Errorf("не удалось проверить sqlite по пути %s: %w", cleanPath, err)
	}
	if fileInfo.IsDir() {
		return nil, fmt.Errorf("ожидался файл sqlite, но получена директория: %s", cleanPath)
	}

	database, err := sql.Open("sqlite", cleanPath)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть sqlite: %w", err)
	}

	if err := database.Ping(); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("не удалось подключиться к sqlite: %w", err)
	}

	if _, err := database.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("не удалось включить внешние ключи sqlite: %w", err)
	}

	if err := applyInitialSchema(database); err != nil {
		_ = database.Close()
		return nil, err
	}

	return database, nil
}

func applyInitialSchema(database *sql.DB) error {
	tx, err := database.Begin()
	if err != nil {
		return fmt.Errorf("не удалось начать миграцию sqlite: %w", err)
	}

	if _, err := tx.Exec(initialSchemaSQL); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("не удалось применить схему sqlite: %w", err)
	}

	if _, err := tx.Exec(
		"INSERT OR IGNORE INTO schema_migrations (version) VALUES (?)",
		initialSchemaVersion,
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("не удалось сохранить версию миграции sqlite: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("не удалось зафиксировать миграцию sqlite: %w", err)
	}

	return nil
}
