package repository

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"map/model"
	"sort"
	"strings"

	"github.com/gosimple/slug"
)

var (
	ErrNotFound            = errors.New("не найдено")
	ErrCategoryNotFound    = errors.New("категория не найдена")
	ErrChildCategoryInUse  = errors.New("подкатегория используется")
	ErrChildCategoryExists = errors.New("подкатегория уже существует")
)

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (repository *SQLiteRepository) ListLocations() ([]model.Location, error) {
	rows, err := repository.db.Query(`
		SELECT
			location.hid,
			location.name,
			location.address,
			location.category_key,
			location.child_category_key,
			location.category_display,
			location.child_category_display,
			location.type_key,
			location.type_display,
			location.manager,
			location.is_partnerships,
			COALESCE(coord.lat, 0),
			COALESCE(coord.lng, 0)
		FROM locations AS location
		LEFT JOIN geo_coords AS coord
			ON coord.owner_type = 'location'
			AND coord.owner_hid = location.hid
			AND coord.coord_kind = 'point'
			AND coord.coord_group = 0
			AND coord.coord_order = 0
		ORDER BY location.category_key, LOWER(location.name)
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	locations := make([]model.Location, 0)
	for rows.Next() {
		var location model.Location
		if err := rows.Scan(
			&location.HID,
			&location.Name,
			&location.Address,
			&location.Category,
			&location.ChildCategory,
			&location.CategoryDisplay,
			&location.ChildCategoryDisplay,
			&location.Type,
			&location.TypeDisplay,
			&location.Manager,
			&location.IsPartnerships,
			&location.Lat,
			&location.Lng,
		); err != nil {
			return nil, err
		}
		locations = append(locations, location)
	}

	return locations, rows.Err()
}

func (repository *SQLiteRepository) GetLocationByHID(hid string) (model.Location, error) {
	var location model.Location
	err := repository.db.QueryRow(`
		SELECT
			location.hid,
			location.name,
			location.address,
			location.category_key,
			location.child_category_key,
			location.category_display,
			location.child_category_display,
			location.type_key,
			location.type_display,
			location.manager,
			location.is_partnerships,
			COALESCE(coord.lat, 0),
			COALESCE(coord.lng, 0)
		FROM locations AS location
		LEFT JOIN geo_coords AS coord
			ON coord.owner_type = 'location'
			AND coord.owner_hid = location.hid
			AND coord.coord_kind = 'point'
			AND coord.coord_group = 0
			AND coord.coord_order = 0
		WHERE location.hid = ?
	`,
		hid,
	).Scan(
		&location.HID,
		&location.Name,
		&location.Address,
		&location.Category,
		&location.ChildCategory,
		&location.CategoryDisplay,
		&location.ChildCategoryDisplay,
		&location.Type,
		&location.TypeDisplay,
		&location.Manager,
		&location.IsPartnerships,
		&location.Lat,
		&location.Lng,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Location{}, ErrNotFound
	}
	return location, err
}

func (repository *SQLiteRepository) CreateLocation(location model.Location) error {
	tx, err := repository.db.Begin()
	if err != nil {
		return err
	}

	if err := upsertLocationTx(tx, location); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (repository *SQLiteRepository) UpdateLocation(oldHID string, location model.Location) error {
	tx, err := repository.db.Begin()
	if err != nil {
		return err
	}

	result, err := tx.Exec(`DELETE FROM locations WHERE hid = ?`, oldHID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	if affected == 0 {
		_ = tx.Rollback()
		return ErrNotFound
	}

	if _, err := tx.Exec(`DELETE FROM geo_coords WHERE owner_type = 'location' AND owner_hid = ?`, oldHID); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := upsertLocationTx(tx, location); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (repository *SQLiteRepository) DeleteLocation(hid string) error {
	tx, err := repository.db.Begin()
	if err != nil {
		return err
	}

	if _, err := tx.Exec(`DELETE FROM geo_coords WHERE owner_type = 'location' AND owner_hid = ?`, hid); err != nil {
		_ = tx.Rollback()
		return err
	}

	result, err := tx.Exec(`DELETE FROM locations WHERE hid = ?`, hid)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	if affected == 0 {
		_ = tx.Rollback()
		return ErrNotFound
	}

	return tx.Commit()
}

func (repository *SQLiteRepository) LoadCategoryConfig() (model.CategoryConfigFile, error) {
	rows, err := repository.db.Query(`
		SELECT key, label, color, sort_order, is_system
		FROM categories
		ORDER BY sort_order, label
	`)
	if err != nil {
		return model.CategoryConfigFile{}, err
	}
	defer rows.Close()

	config := model.CategoryConfigFile{Categories: make([]model.CategoryConfig, 0)}
	for rows.Next() {
		var category model.CategoryConfig
		if err := rows.Scan(&category.Key, &category.Label, &category.Color, &category.SortOrder, &category.IsSystem); err != nil {
			return model.CategoryConfigFile{}, err
		}
		category.Children, err = repository.loadCategoryChildren(category.Key)
		if err != nil {
			return model.CategoryConfigFile{}, err
		}
		config.Categories = append(config.Categories, category)
	}

	return config, rows.Err()
}

func (repository *SQLiteRepository) CreateChildCategory(categoryKey string, child model.CategoryChild) (model.CategoryChild, error) {
	if !repository.categoryExists(categoryKey) {
		return model.CategoryChild{}, ErrCategoryNotFound
	}

	_, err := repository.db.Exec(`
		INSERT INTO category_children (category_key, key, label)
		VALUES (?, ?, ?)
	`, categoryKey, child.Key, child.Label)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return model.CategoryChild{}, ErrChildCategoryExists
		}
		return model.CategoryChild{}, err
	}

	return child, nil
}

func (repository *SQLiteRepository) UpdateChildCategory(categoryKey string, oldChildKey string, child model.CategoryChild) (model.CategoryChild, error) {
	if !repository.categoryExists(categoryKey) {
		return model.CategoryChild{}, ErrCategoryNotFound
	}

	tx, err := repository.db.Begin()
	if err != nil {
		return model.CategoryChild{}, err
	}

	result, err := tx.Exec(`
		UPDATE category_children
		SET key = ?, label = ?, updated_at = CURRENT_TIMESTAMP
		WHERE category_key = ? AND key = ?
	`, child.Key, child.Label, categoryKey, oldChildKey)
	if err != nil {
		_ = tx.Rollback()
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return model.CategoryChild{}, ErrChildCategoryExists
		}
		return model.CategoryChild{}, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return model.CategoryChild{}, err
	}
	if affected == 0 {
		_ = tx.Rollback()
		return model.CategoryChild{}, ErrNotFound
	}

	if categoryKey == "bonetsky" {
		if _, err := tx.Exec(`
			UPDATE locations
			SET
				type_key = ?,
				type_display = ?,
				updated_at = CURRENT_TIMESTAMP
			WHERE category_key = ? AND type_key = ?
		`, child.Key, child.Label, categoryKey, oldChildKey); err != nil {
			_ = tx.Rollback()
			return model.CategoryChild{}, err
		}
	} else {
		if _, err := tx.Exec(`
			UPDATE locations
			SET
				child_category_key = ?,
				child_category_display = ?,
				updated_at = CURRENT_TIMESTAMP
			WHERE category_key = ? AND child_category_key = ?
		`, child.Key, child.Label, categoryKey, oldChildKey); err != nil {
			_ = tx.Rollback()
			return model.CategoryChild{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return model.CategoryChild{}, err
	}

	return child, nil
}

func (repository *SQLiteRepository) DeleteChildCategory(categoryKey string, childKey string) error {
	if !repository.categoryExists(categoryKey) {
		return ErrCategoryNotFound
	}

	var inUse bool
	var err error
	if categoryKey == "bonetsky" {
		err = repository.db.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM locations WHERE category_key = ? AND type_key = ?
			)
		`, categoryKey, childKey).Scan(&inUse)
	} else {
		err = repository.db.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM locations WHERE category_key = ? AND child_category_key = ?
			)
		`, categoryKey, childKey).Scan(&inUse)
	}
	if err != nil {
		return err
	}
	if inUse {
		return ErrChildCategoryInUse
	}

	result, err := repository.db.Exec(`
		DELETE FROM category_children
		WHERE category_key = ? AND key = ?
	`, categoryKey, childKey)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (repository *SQLiteRepository) ListGeoItems(collection string) ([]model.GeoItem, error) {
	tableName, err := collectionTable(collection)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`
		SELECT
			base.hid,
			base.title,
			base.population,
			COALESCE(coord.lat, 0),
			COALESCE(coord.lng, 0)
		FROM %s AS base
		LEFT JOIN geo_coords AS coord
			ON coord.owner_type = ?
			AND coord.owner_hid = base.hid
			AND coord.coord_kind = 'point'
			AND coord.coord_group = 0
			AND coord.coord_order = 0
		ORDER BY base.title
	`, tableName)

	rows, err := repository.db.Query(query, collectionOwnerType(collection))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.GeoItem, 0)
	for rows.Next() {
		var item model.GeoItem
		if err := rows.Scan(&item.HID, &item.Title, &item.Population, &item.Lat, &item.Lng); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (repository *SQLiteRepository) GetGeoCoords(collection string, hid string) ([][][]float64, error) {
	ownerType := collectionOwnerType(collection)
	if ownerType == "" {
		return nil, ErrNotFound
	}

	rows, err := repository.db.Query(`
		SELECT coord_group, coord_order, lat, lng
		FROM geo_coords
		WHERE owner_type = ? AND owner_hid = ? AND coord_kind = 'polygon'
		ORDER BY coord_group, coord_order
	`, ownerType, hid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	grouped := map[int][][]float64{}
	order := make([]int, 0)

	for rows.Next() {
		var groupIndex int
		var coordOrder int
		var lat float64
		var lng float64
		if err := rows.Scan(&groupIndex, &coordOrder, &lat, &lng); err != nil {
			return nil, err
		}
		if _, exists := grouped[groupIndex]; !exists {
			grouped[groupIndex] = make([][]float64, 0)
			order = append(order, groupIndex)
		}
		grouped[groupIndex] = append(grouped[groupIndex], []float64{lat, lng})
	}

	sort.Ints(order)
	result := make([][][]float64, 0, len(order))
	for _, groupIndex := range order {
		result = append(result, grouped[groupIndex])
	}
	return result, rows.Err()
}

func (repository *SQLiteRepository) loadCategoryChildren(categoryKey string) ([]model.CategoryChild, error) {
	rows, err := repository.db.Query(`
		SELECT key, label
		FROM category_children
		WHERE category_key = ?
		ORDER BY label
	`, categoryKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	children := make([]model.CategoryChild, 0)
	for rows.Next() {
		var child model.CategoryChild
		if err := rows.Scan(&child.Key, &child.Label); err != nil {
			return nil, err
		}
		children = append(children, child)
	}

	return children, rows.Err()
}

func (repository *SQLiteRepository) categoryExists(categoryKey string) bool {
	var exists bool
	_ = repository.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM categories WHERE key = ?)`, categoryKey).Scan(&exists)
	return exists
}

func upsertLocationTx(tx *sql.Tx, location model.Location) error {
	if _, err := tx.Exec(`
		INSERT INTO locations (
			hid, name, address, category_key, child_category_key,
			category_display, child_category_display, type_key, type_display,
			manager, is_partnerships, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`, location.HID, location.Name, location.Address, location.Category, location.ChildCategory,
		location.CategoryDisplay, location.ChildCategoryDisplay, location.Type, location.TypeDisplay,
		location.Manager, location.IsPartnerships,
	); err != nil {
		return err
	}

	_, err := tx.Exec(`
		INSERT INTO geo_coords (
			owner_type, owner_hid, coord_kind, coord_group, coord_order, lat, lng
		) VALUES ('location', ?, 'point', 0, 0, ?, ?)
	`, location.HID, location.Lat, location.Lng)
	return err
}

func GenerateLocationHID(name string) string {
	normalized := strings.TrimSpace(strings.ToLower(name))
	sum := md5.Sum([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

func SlugifyKey(value string) string {
	result := strings.Trim(slug.MakeLang(strings.TrimSpace(value), "ru"), "-")
	result = strings.ReplaceAll(result, "-", "_")
	if result == "" {
		return "child_category"
	}
	return result
}

func collectionTable(collection string) (string, error) {
	switch collection {
	case "cities":
		return "cities", nil
	case "districts":
		return "districts", nil
	case "regions":
		return "regions", nil
	default:
		return "", ErrNotFound
	}
}

func collectionOwnerType(collection string) string {
	switch collection {
	case "cities":
		return "city"
	case "districts":
		return "district"
	case "regions":
		return "region"
	default:
		return ""
	}
}
