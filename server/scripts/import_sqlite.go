package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"map/repository"
	dbsqlite "map/sqlite"

	"github.com/joho/godotenv"
)

type importOptions struct {
	Mode       string
	ImportMode string
}

func main() {
	options := parseArgs()
	loadImportEnv(options.Mode)

	sqlitePath, cleanup, err := resolveSQLitePath(options.ImportMode)
	if err != nil {
		log.Fatal(err)
	}
	if cleanup != nil {
		defer cleanup()
	}

	database, err := dbsqlite.OpenAndMigrate(sqlitePath)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	if err := prepareDatabaseForImport(database, options.ImportMode); err != nil {
		log.Fatal(err)
	}

	if err := repository.BootstrapIfEmpty(database, readImportDataFile); err != nil {
		log.Fatal(err)
	}

	printImportStats(database, sqlitePath, options.ImportMode)
}

func parseArgs() importOptions {
	modeFlag := flag.String("mode", "prod", "режим env: dev или prod")
	importModeFlag := flag.String("import-mode", "overwrite", "режим импорта: initial, overwrite, dry-run")
	flag.Parse()

	mode := strings.TrimSpace(*modeFlag)
	if mode != "dev" && mode != "prod" {
		log.Fatalf("неверный режим %q, используй dev или prod", mode)
	}

	importMode := strings.TrimSpace(*importModeFlag)
	switch importMode {
	case "initial", "overwrite", "dry-run":
	default:
		log.Fatalf("неверный import-mode %q, используй initial, overwrite или dry-run", importMode)
	}

	return importOptions{
		Mode:       mode,
		ImportMode: importMode,
	}
}

func loadImportEnv(mode string) {
	_ = godotenv.Load(".env")
	if mode == "dev" {
		_ = godotenv.Load(".env.development")
		return
	}
	_ = godotenv.Load(".env.production")
}

func readImportDataFile(name string) ([]byte, error) {
	fullPath := filepath.Join("..", "data", name)
	payload, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать %s: %w", fullPath, err)
	}
	return payload, nil
}

func resolveSQLitePath(importMode string) (string, func(), error) {
	if importMode == "dry-run" {
		tempDir, err := os.MkdirTemp("", "map-import-dry-run-*")
		if err != nil {
			return "", nil, fmt.Errorf("не удалось создать временную директорию dry-run: %w", err)
		}

		tempPath := filepath.Join(tempDir, "dry-run.sqlite")
		return tempPath, func() {
			_ = os.RemoveAll(tempDir)
		}, nil
	}

	sqlitePath := strings.TrimSpace(os.Getenv("SQLITE_PATH"))
	if sqlitePath == "" {
		return "", nil, fmt.Errorf("переменная SQLITE_PATH обязательна")
	}

	return sqlitePath, nil, nil
}

func prepareDatabaseForImport(database *sql.DB, importMode string) error {
	switch importMode {
	case "initial":
		isEmpty, err := isDatabaseEmpty(database)
		if err != nil {
			return err
		}
		if !isEmpty {
			return fmt.Errorf("sqlite уже содержит данные; для полной перезаписи используй import-mode=overwrite")
		}
		return nil
	case "overwrite", "dry-run":
		return resetSQLite(database)
	default:
		return fmt.Errorf("неподдерживаемый режим импорта: %s", importMode)
	}
}

func isDatabaseEmpty(database *sql.DB) (bool, error) {
	queries := []string{
		`SELECT COUNT(*) FROM categories`,
		`SELECT COUNT(*) FROM category_children`,
		`SELECT COUNT(*) FROM locations`,
		`SELECT COUNT(*) FROM regions`,
		`SELECT COUNT(*) FROM cities`,
		`SELECT COUNT(*) FROM districts`,
		`SELECT COUNT(*) FROM geo_coords`,
	}

	for _, query := range queries {
		var count int
		if err := database.QueryRow(query).Scan(&count); err != nil {
			return false, fmt.Errorf("не удалось проверить заполненность sqlite: %w", err)
		}
		if count > 0 {
			return false, nil
		}
	}

	return true, nil
}

func resetSQLite(database *sql.DB) error {
	tx, err := database.Begin()
	if err != nil {
		return fmt.Errorf("не удалось начать очистку sqlite: %w", err)
	}

	statements := []string{
		`DELETE FROM geo_coords;`,
		`DELETE FROM locations;`,
		`DELETE FROM category_children;`,
		`DELETE FROM categories;`,
		`DELETE FROM districts;`,
		`DELETE FROM cities;`,
		`DELETE FROM regions;`,
		`DELETE FROM sqlite_sequence;`,
	}

	for _, statement := range statements {
		if _, err := tx.Exec(statement); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("не удалось очистить sqlite: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("не удалось завершить очистку sqlite: %w", err)
	}

	return nil
}

func printImportStats(database *sql.DB, sqlitePath string, importMode string) {
	stats := []struct {
		label string
		query string
	}{
		{label: "Категории", query: `SELECT COUNT(*) FROM categories`},
		{label: "Подкатегории", query: `SELECT COUNT(*) FROM category_children`},
		{label: "Локации", query: `SELECT COUNT(*) FROM locations`},
		{label: "Регионы", query: `SELECT COUNT(*) FROM regions`},
		{label: "Города", query: `SELECT COUNT(*) FROM cities`},
		{label: "Районы", query: `SELECT COUNT(*) FROM districts`},
		{label: "Координаты", query: `SELECT COUNT(*) FROM geo_coords`},
	}

	if importMode == "dry-run" {
		log.Printf("dry-run импорт успешно проверен")
	} else {
		log.Printf("импорт в sqlite завершён")
		log.Printf("sqlite: %s", sqlitePath)
	}
	for _, stat := range stats {
		var count int
		if err := database.QueryRow(stat.query).Scan(&count); err != nil {
			log.Printf("%s: ошибка чтения статистики: %v", stat.label, err)
			continue
		}
		log.Printf("%s: %d", stat.label, count)
	}
}
