package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"map/model"
	"strings"
)

type geoSeedItem struct {
	HID        string        `json:"hid"`
	Title      string        `json:"title"`
	Population int           `json:"population"`
	Lat        float64       `json:"lat"`
	Lng        float64       `json:"lng"`
	Coords     [][][]float64 `json:"coords"`
}

func BootstrapIfEmpty(db *sql.DB, readFile func(string) ([]byte, error)) error {
	repository := NewSQLiteRepository(db)

	if err := bootstrapCategoriesIfEmpty(db, readFile); err != nil {
		return err
	}
	if err := bootstrapLocationsIfEmpty(db, readFile); err != nil {
		return err
	}
	if err := bootstrapGeoIfEmpty(repository, db, readFile, "regions"); err != nil {
		return err
	}
	if err := bootstrapGeoIfEmpty(repository, db, readFile, "cities"); err != nil {
		return err
	}
	if err := bootstrapGeoIfEmpty(repository, db, readFile, "districts"); err != nil {
		return err
	}

	return nil
}

func bootstrapCategoriesIfEmpty(db *sql.DB, readFile func(string) ([]byte, error)) error {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM categories`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	payload, err := readFile("location_categories.json")
	if err != nil {
		return fmt.Errorf("не удалось прочитать bootstrap JSON категорий: %w", err)
	}

	var config model.CategoryConfigFile
	if err := json.Unmarshal(payload, &config); err != nil {
		return fmt.Errorf("не удалось разобрать bootstrap JSON категорий: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	for _, category := range config.Categories {
		if _, err := tx.Exec(`
			INSERT INTO categories (key, label, color, sort_order, is_system)
			VALUES (?, ?, ?, ?, ?)
		`, category.Key, category.Label, category.Color, category.SortOrder, category.IsSystem); err != nil {
			_ = tx.Rollback()
			return err
		}

		for _, child := range category.Children {
			if _, err := tx.Exec(`
				INSERT INTO category_children (category_key, key, label)
				VALUES (?, ?, ?)
			`, category.Key, child.Key, child.Label); err != nil {
				_ = tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit()
}

func bootstrapLocationsIfEmpty(db *sql.DB, readFile func(string) ([]byte, error)) error {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM locations`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	payload, err := readFile("locations.json")
	if err != nil {
		return fmt.Errorf("не удалось прочитать bootstrap JSON локаций: %w", err)
	}

	var wrapped model.LocationsFile
	if err := json.Unmarshal(payload, &wrapped); err != nil || wrapped.Locations == nil {
		var plain []model.Location
		if err := json.Unmarshal(payload, &plain); err != nil {
			return fmt.Errorf("не удалось разобрать bootstrap JSON локаций: %w", err)
		}
		wrapped.Locations = plain
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	for _, location := range wrapped.Locations {
		if strings.TrimSpace(location.HID) == "" {
			location.HID = GenerateLocationHID(location.Name)
		}
		if _, err := tx.Exec(`
			INSERT INTO locations (
				hid, name, address, category_key, child_category_key,
				category_display, child_category_display, type_key, type_display,
				manager, is_partnerships
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, location.HID, location.Name, location.Address, location.Category, location.ChildCategory,
			location.CategoryDisplay, location.ChildCategoryDisplay, location.Type, location.TypeDisplay,
			location.Manager, location.IsPartnerships,
		); err != nil {
			_ = tx.Rollback()
			return err
		}

		lat, lng := normalizeLatLng(location.Lat, location.Lng)
		if _, err := tx.Exec(`
			INSERT INTO geo_coords (owner_type, owner_hid, coord_kind, coord_group, coord_order, lat, lng)
			VALUES ('location', ?, 'point', 0, 0, ?, ?)
		`, location.HID, lat, lng); err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func bootstrapGeoIfEmpty(repository *SQLiteRepository, db *sql.DB, readFile func(string) ([]byte, error), collection string) error {
	tableName, err := collectionTable(collection)
	if err != nil {
		return err
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ` + tableName).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	payload, err := readFile(collection + ".json")
	if err != nil {
		return fmt.Errorf("не удалось прочитать bootstrap JSON %s: %w", collection, err)
	}

	var items []geoSeedItem
	if err := json.Unmarshal(payload, &items); err != nil {
		return fmt.Errorf("не удалось разобрать bootstrap JSON %s: %w", collection, err)
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	ownerType := collectionOwnerType(collection)
	for _, item := range items {
		if strings.TrimSpace(item.HID) == "" {
			item.HID = GenerateLocationHID(item.Title)
		}

		centerLat, centerLng := normalizeLatLng(item.Lat, item.Lng)
		if _, err := tx.Exec(
			fmt.Sprintf(`INSERT INTO %s (hid, title, population) VALUES (?, ?, ?)`, tableName),
			item.HID, item.Title, item.Population,
		); err != nil {
			_ = tx.Rollback()
			return err
		}

		if _, err := tx.Exec(`
			INSERT INTO geo_coords (owner_type, owner_hid, coord_kind, coord_group, coord_order, lat, lng)
			VALUES (?, ?, 'point', 0, 0, ?, ?)
		`, ownerType, item.HID, centerLat, centerLng); err != nil {
			_ = tx.Rollback()
			return err
		}

		for groupIndex, ring := range item.Coords {
			for coordIndex, pair := range ring {
				if len(pair) < 2 {
					continue
				}
				lat, lng := normalizeLatLng(pair[0], pair[1])
				if _, err := tx.Exec(`
					INSERT INTO geo_coords (owner_type, owner_hid, coord_kind, coord_group, coord_order, lat, lng)
					VALUES (?, ?, 'polygon', ?, ?, ?, ?)
				`, ownerType, item.HID, groupIndex, coordIndex, lat, lng); err != nil {
					_ = tx.Rollback()
					return err
				}
			}
		}
	}

	return tx.Commit()
}

func normalizeLatLng(lat float64, lng float64) (float64, float64) {
	if absFloat(lat) > 90 && absFloat(lng) <= 90 {
		return lng, lat
	}
	if lat >= 69 && lat <= 81 && lng >= 39 && lng <= 44 {
		return lng, lat
	}
	return lat, lng
}

func absFloat(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}
