package api

import (
	"errors"
	"map/repository"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

func (api *LocationsAPI) NewDataRouter() chi.Router {
	router := chi.NewRouter()
	router.Group(func(group chi.Router) {
		group.Use(api.tokenMiddleware)
		group.Get("/data/locations", api.handleDataLocations)
		group.Get("/data/cities", api.handleDataCities)
		group.Get("/data/districts", api.handleDataDistricts)
		group.Get("/data/regions", api.handleDataRegions)
		group.Get("/data/location_categories", api.handleDataLocationCategories)
		group.Get("/data/{collection}/{hid}/coords", api.handleDataCoords)
	})
	return router
}

func (api *LocationsAPI) handleDataLocations(w http.ResponseWriter, r *http.Request) {
	locations, err := api.repository.ListLocations()
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{Code: "locations_read_failed", Message: err.Error()})
		return
	}
	render.JSON(w, r, map[string]any{"locations": locations})
}

func (api *LocationsAPI) handleDataLocationCategories(w http.ResponseWriter, r *http.Request) {
	config, err := api.repository.LoadCategoryConfig()
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{Code: "config_read_failed", Message: err.Error()})
		return
	}
	render.JSON(w, r, config)
}

func (api *LocationsAPI) handleDataCities(w http.ResponseWriter, r *http.Request) {
	api.serveGeoCollection(w, r, "cities")
}

func (api *LocationsAPI) handleDataDistricts(w http.ResponseWriter, r *http.Request) {
	api.serveGeoCollection(w, r, "districts")
}

func (api *LocationsAPI) handleDataRegions(w http.ResponseWriter, r *http.Request) {
	api.serveGeoCollection(w, r, "regions")
}

func (api *LocationsAPI) serveGeoCollection(w http.ResponseWriter, r *http.Request, collection string) {
	items, err := api.repository.ListGeoItems(collection)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{Code: "data_read_failed", Message: err.Error()})
		return
	}
	render.JSON(w, r, items)
}

func (api *LocationsAPI) handleDataCoords(w http.ResponseWriter, r *http.Request) {
	collection := strings.ToLower(strings.TrimSpace(chi.URLParam(r, "collection")))
	hid := strings.TrimSpace(chi.URLParam(r, "hid"))

	if hid == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse{Code: "invalid_hid", Message: "идентификатор обязателен"})
		return
	}

	coords, err := api.repository.GetGeoCoords(collection, hid)
	if errors.Is(err, repository.ErrNotFound) {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, errorResponse{Code: "not_found", Message: "ничего не найдено"})
		return
	}
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{Code: "data_read_failed", Message: err.Error()})
		return
	}

	render.JSON(w, r, coords)
}
