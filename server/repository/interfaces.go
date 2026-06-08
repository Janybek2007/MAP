package repository

import "map/model"

type LocationRepository interface {
	ListLocations() ([]model.Location, error)
	GetLocationByHID(hid string) (model.Location, error)
	CreateLocation(location model.Location) error
	UpdateLocation(oldHID string, location model.Location) error
	DeleteLocation(hid string) error
}

type CategoryRepository interface {
	LoadCategoryConfig() (model.CategoryConfigFile, error)
	CreateChildCategory(categoryKey string, child model.CategoryChild) (model.CategoryChild, error)
	UpdateChildCategory(categoryKey string, oldChildKey string, child model.CategoryChild) (model.CategoryChild, error)
	DeleteChildCategory(categoryKey string, childKey string) error
}

type GeoRepository interface {
	ListGeoItems(collection string) ([]model.GeoItem, error)
	GetGeoCoords(collection string, hid string) ([][][]float64, error)
}

type Repository interface {
	LocationRepository
	CategoryRepository
	GeoRepository
}
