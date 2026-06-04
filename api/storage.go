package api

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/gosimple/slug"
)

type Storage struct {
	locationsPath string
	configPath    string
	mu            sync.Mutex
}

func NewStorage(dataDir string) *Storage {
	return &Storage{
		locationsPath: filepath.Join(dataDir, "locations.json"),
		configPath:    filepath.Join(dataDir, "location_categories.json"),
	}
}

func (storage *Storage) LoadLocations() ([]Location, error) {
	storage.mu.Lock()
	defer storage.mu.Unlock()

	payload, err := os.ReadFile(storage.locationsPath)
	if err != nil {
		return nil, err
	}

	var wrapped locationsFile
	if err := json.Unmarshal(payload, &wrapped); err == nil && wrapped.Locations != nil {
		return wrapped.Locations, nil
	}

	var plain []Location
	if err := json.Unmarshal(payload, &plain); err != nil {
		return nil, err
	}

	return plain, nil
}

func (storage *Storage) SaveLocations(locations []Location) error {
	storage.mu.Lock()
	defer storage.mu.Unlock()

	sort.SliceStable(locations, func(i int, j int) bool {
		if locations[i].Category != locations[j].Category {
			return locations[i].Category < locations[j].Category
		}
		return strings.ToLower(locations[i].Name) < strings.ToLower(locations[j].Name)
	})

	if err := backupIfExists(storage.locationsPath); err != nil {
		return err
	}

	payload, err := json.MarshalIndent(locationsFile{Locations: locations}, "", "\t")
	if err != nil {
		return err
	}

	tmpPath := storage.locationsPath + ".tmp"
	if err := os.WriteFile(tmpPath, payload, 0o644); err != nil {
		return err
	}

	return os.Rename(tmpPath, storage.locationsPath)
}

func (storage *Storage) LoadCategoryConfig() (CategoryConfigFile, error) {
	storage.mu.Lock()
	defer storage.mu.Unlock()

	payload, err := os.ReadFile(storage.configPath)
	if err != nil {
		return CategoryConfigFile{}, err
	}

	var config CategoryConfigFile
	if err := json.Unmarshal(payload, &config); err != nil {
		return CategoryConfigFile{}, err
	}

	return config, nil
}

func (storage *Storage) SaveCategoryConfig(config CategoryConfigFile) error {
	storage.mu.Lock()
	defer storage.mu.Unlock()

	sort.SliceStable(config.Categories, func(i int, j int) bool {
		if config.Categories[i].SortOrder != config.Categories[j].SortOrder {
			return config.Categories[i].SortOrder < config.Categories[j].SortOrder
		}
		return config.Categories[i].Label < config.Categories[j].Label
	})

	for index := range config.Categories {
		sort.SliceStable(config.Categories[index].Children, func(i int, j int) bool {
			return strings.ToLower(config.Categories[index].Children[i].Label) < strings.ToLower(config.Categories[index].Children[j].Label)
		})
	}

	if err := backupIfExists(storage.configPath); err != nil {
		return err
	}

	payload, err := json.MarshalIndent(config, "", "\t")
	if err != nil {
		return err
	}

	tmpPath := storage.configPath + ".tmp"
	if err := os.WriteFile(tmpPath, payload, 0o644); err != nil {
		return err
	}

	return os.Rename(tmpPath, storage.configPath)
}

func backupIfExists(filePath string) error {
	payload, err := os.ReadFile(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	return os.WriteFile(filePath+".bak", payload, 0o644)
}

func generateLocationHID(name string) string {
	normalized := strings.TrimSpace(strings.ToLower(name))
	sum := md5.Sum([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

func slugifyKey(value string) string {
	result := strings.Trim(slug.MakeLang(strings.TrimSpace(value), "ru"), "-")
	result = strings.ReplaceAll(result, "-", "_")
	if result == "" {
		return "child_category"
	}
	return result
}
