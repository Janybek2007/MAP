package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

func (api *LocationsAPI) NewDataRouter() chi.Router {
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(api.tokenMiddleware)
		r.Get("/data/locations", api.handleDataLocations)
		r.Get("/data/cities", api.handleDataCities)
		r.Get("/data/districts", api.handleDataDistricts)
		r.Get("/data/regions", api.handleDataRegions)
		r.Get("/data/location_categories", api.handleDataLocationCategories)
		r.Get("/data/{collection}/{hid}/coords", api.handleDataCoords)
	})
	return r
}

func (api *LocationsAPI) handleDataLocations(w http.ResponseWriter, r *http.Request) {
	locations, err := api.storage.LoadLocations()
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{Code: "locations_read_failed", Message: err.Error()})
		return
	}
	render.JSON(w, r, map[string]any{"locations": locations})
}

func (api *LocationsAPI) handleDataLocationCategories(w http.ResponseWriter, r *http.Request) {
	config, err := api.storage.LoadCategoryConfig()
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{Code: "config_read_failed", Message: err.Error()})
		return
	}
	render.JSON(w, r, config)
}

func (api *LocationsAPI) handleDataCities(w http.ResponseWriter, r *http.Request) {
	api.serveDataFileStripped(w, r, "cities.json")
}

func (api *LocationsAPI) handleDataDistricts(w http.ResponseWriter, r *http.Request) {
	api.serveDataFileStripped(w, r, "districts.json")
}

func (api *LocationsAPI) handleDataRegions(w http.ResponseWriter, r *http.Request) {
	api.serveDataFileStripped(w, r, "regions.json")
}

func (api *LocationsAPI) serveDataFileStripped(w http.ResponseWriter, r *http.Request, name string) {
	raw, err := api.readFile(name)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{Code: "data_read_failed", Message: err.Error()})
		return
	}

	var items []map[string]any
	if err := json.Unmarshal(raw, &items); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{Code: "data_parse_failed", Message: "invalid json in data file"})
		return
	}

	for i := range items {
		delete(items[i], "coords")
	}

	render.JSON(w, r, items)
}

func (api *LocationsAPI) handleDataCoords(w http.ResponseWriter, r *http.Request) {
	collection := strings.ToLower(strings.TrimSpace(chi.URLParam(r, "collection")))
	hid := strings.TrimSpace(chi.URLParam(r, "hid"))

	fileName := collection + ".json"
	if fileName != "districts.json" && fileName != "cities.json" && fileName != "regions.json" {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, errorResponse{Code: "not_found", Message: "not found"})
		return
	}

	if hid == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse{Code: "invalid_hid", Message: "hid is required"})
		return
	}

	raw, err := api.readFile(fileName)
	if err != nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, errorResponse{Code: "not_found", Message: "not found"})
		return
	}

	var items []map[string]any
	if err := json.Unmarshal(raw, &items); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{Code: "data_parse_failed", Message: "invalid json in data file"})
		return
	}

	for _, item := range items {
		if itemHID, _ := item["hid"].(string); itemHID == hid {
			coords := item["coords"]
			if coords == nil {
				coords = []any{}
			}
			render.JSON(w, r, coords)
			return
		}
	}

	render.Status(r, http.StatusNotFound)
	render.JSON(w, r, errorResponse{Code: "not_found", Message: "not found"})
}
