package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

const (
	tokenHeader         = "X-Resource-Token"
	tokenEndpointHeader = "X-Token"
	nextTokenHeader     = "X-Next-Token"
	nextTokenTTLHeader  = "X-Next-Token-Expires-At"
)

type LocationsAPI struct {
	router               chi.Router
	storage              *Storage
	tokenManager         *TokenManager
	readFile             func(string) ([]byte, error)
	protectTokenEndpoint bool
}

func NewLocationsAPI(router chi.Router, storage *Storage, tokenManager *TokenManager, readFile func(string) ([]byte, error), protectTokenEndpoint bool) *LocationsAPI {
	return &LocationsAPI{
		router:               router,
		storage:              storage,
		tokenManager:         tokenManager,
		readFile:             readFile,
		protectTokenEndpoint: protectTokenEndpoint,
	}
}

func RegisterRoute(router chi.Router, dataDir string, apiKey string, readFile func(string) ([]byte, error), protectTokenEndpoint bool) *LocationsAPI {
	api := NewLocationsAPI(router, NewStorage(dataDir), NewTokenManager(apiKey), readFile, protectTokenEndpoint)
	api.register()
	return api
}

func (api *LocationsAPI) IssueTokenEndpointToken() (string, int64, error) {
	response, err := api.tokenManager.IssueTokenEndpoint()
	if err != nil {
		return "", 0, err
	}
	return response.Token, response.ExpiresAt, nil
}

func (api *LocationsAPI) register() {
	api.router.Route("/api", func(router chi.Router) {
		router.Post("/tokens", api.handleCreateToken)

		router.Group(func(protectedRouter chi.Router) {
			protectedRouter.Use(api.tokenMiddleware)
			protectedRouter.Get("/locations", api.handleListLocations)
			protectedRouter.Get("/locations/{hid}", api.handleGetLocation)
			protectedRouter.Get("/location-config", api.handleGetConfig)
			protectedRouter.Post("/locations", api.handleCreateLocation)
			protectedRouter.Put("/locations/{hid}", api.handleUpdateLocation)
			protectedRouter.Delete("/locations/{hid}", api.handleDeleteLocation)
			protectedRouter.Post("/location-config/{category}/children", api.handleCreateChildCategory)
			protectedRouter.Put("/location-config/{category}/children/{child}", api.handleUpdateChildCategory)
			protectedRouter.Delete("/location-config/{category}/children/{child}", api.handleDeleteChildCategory)
		})
	})
}

func (api *LocationsAPI) tokenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimSpace(r.Header.Get(tokenHeader))
		if token == "" {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, errorResponse{Code: "missing_token", Message: "resource token is required"})
			return
		}

		if err := api.tokenManager.Validate(token, r.Method, r.URL.Path); err != nil {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, errorResponse{Code: "invalid_token", Message: err.Error()})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (api *LocationsAPI) handleCreateToken(w http.ResponseWriter, r *http.Request) {
	if api.protectTokenEndpoint {
		token := strings.TrimSpace(r.Header.Get(tokenEndpointHeader))
		if token == "" {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, errorResponse{Code: "missing_token_endpoint_token", Message: "token endpoint token is required"})
			return
		}

		if err := api.tokenManager.ValidateTokenEndpoint(token); err != nil {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, errorResponse{Code: "invalid_token_endpoint_token", Message: err.Error()})
			return
		}

		if !api.setNextTokenEndpointHeaders(w) {
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, errorResponse{Code: "token_endpoint_issue_failed", Message: "failed to issue next token endpoint token"})
			return
		}
	}

	var raw json.RawMessage
	if err := render.DecodeJSON(r.Body, &raw); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse{Code: "invalid_body", Message: "invalid token request body"})
		return
	}

	if len(raw) > 0 && raw[0] == '[' {
		var requests []tokenRequest
		if err := json.Unmarshal(raw, &requests); err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, errorResponse{Code: "invalid_body", Message: "invalid token request body"})
			return
		}
		responses := make([]tokenResponse, len(requests))
		for i, req := range requests {
			resp, err := api.tokenManager.Issue(req.Method, req.URL)
			if err != nil {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, errorResponse{Code: "invalid_token_request", Message: err.Error()})
				return
			}
			responses[i] = resp
		}
		render.JSON(w, r, responses)
		return
	}

	var request tokenRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse{Code: "invalid_body", Message: "invalid token request body"})
		return
	}
	response, err := api.tokenManager.Issue(request.Method, request.URL)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse{Code: "invalid_token_request", Message: err.Error()})
		return
	}
	render.JSON(w, r, response)
}

func (api *LocationsAPI) setNextTokenEndpointHeaders(w http.ResponseWriter) bool {
	token, expiresAt, err := api.IssueTokenEndpointToken()
	if err != nil {
		return false
	}
	w.Header().Set(nextTokenHeader, token)
	w.Header().Set(nextTokenTTLHeader, strconv.FormatInt(expiresAt, 10))
	return true
}

func (api *LocationsAPI) handleListLocations(w http.ResponseWriter, r *http.Request) {
	locations, err := api.storage.LoadLocations()
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{Code: "locations_read_failed", Message: err.Error()})
		return
	}

	render.JSON(w, r, map[string]any{"locations": locations})
}

func (api *LocationsAPI) handleGetLocation(w http.ResponseWriter, r *http.Request) {
	hid := chi.URLParam(r, "hid")
	location, err := api.findLocationByHID(hid)
	if err != nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, errorResponse{Code: "location_not_found", Message: "location not found"})
		return
	}

	render.JSON(w, r, location)
}

func (api *LocationsAPI) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	config, err := api.storage.LoadCategoryConfig()
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{Code: "config_read_failed", Message: err.Error()})
		return
	}

	render.JSON(w, r, config)
}

func (api *LocationsAPI) handleCreateLocation(w http.ResponseWriter, r *http.Request) {
	var location Location
	if err := render.DecodeJSON(r.Body, &location); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse{Code: "invalid_body", Message: "invalid location body"})
		return
	}

	locations, config, err := api.loadLocationsAndConfig()
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{Code: "storage_error", Message: err.Error()})
		return
	}

	normalized, fieldErrors := normalizeAndValidateLocation(location, config)
	if len(fieldErrors) > 0 {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse{Code: "validation_failed", Message: "location validation failed", Fields: fieldErrors})
		return
	}

	normalized.HID = generateLocationHID(normalized.Name)
	if hasLocationHID(locations, normalized.HID, "") {
		render.Status(r, http.StatusConflict)
		render.JSON(w, r, errorResponse{Code: "hid_conflict", Message: "location with same generated hid already exists"})
		return
	}

	locations = append(locations, normalized)
	if err := api.storage.SaveLocations(locations); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{Code: "locations_write_failed", Message: err.Error()})
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, normalized)
}

func (api *LocationsAPI) handleUpdateLocation(w http.ResponseWriter, r *http.Request) {
	hid := chi.URLParam(r, "hid")
	if strings.TrimSpace(hid) == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse{Code: "invalid_hid", Message: "hid is required"})
		return
	}

	var location Location
	if err := render.DecodeJSON(r.Body, &location); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse{Code: "invalid_body", Message: "invalid location body"})
		return
	}

	locations, config, err := api.loadLocationsAndConfig()
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{Code: "storage_error", Message: err.Error()})
		return
	}

	index := slices.IndexFunc(locations, func(item Location) bool {
		return item.HID == hid
	})
	if index < 0 {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, errorResponse{Code: "location_not_found", Message: "location not found"})
		return
	}

	normalized, fieldErrors := normalizeAndValidateLocation(location, config)
	if len(fieldErrors) > 0 {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse{Code: "validation_failed", Message: "location validation failed", Fields: fieldErrors})
		return
	}

	normalized.HID = generateLocationHID(normalized.Name)
	if hasLocationHID(locations, normalized.HID, hid) {
		render.Status(r, http.StatusConflict)
		render.JSON(w, r, errorResponse{Code: "hid_conflict", Message: "location with same generated hid already exists"})
		return
	}

	locations[index] = normalized
	if err := api.storage.SaveLocations(locations); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{Code: "locations_write_failed", Message: err.Error()})
		return
	}

	render.JSON(w, r, normalized)
}

func (api *LocationsAPI) handleDeleteLocation(w http.ResponseWriter, r *http.Request) {
	hid := chi.URLParam(r, "hid")
	if strings.TrimSpace(hid) == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse{Code: "invalid_hid", Message: "hid is required"})
		return
	}

	locations, err := api.storage.LoadLocations()
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{Code: "locations_read_failed", Message: err.Error()})
		return
	}

	index := slices.IndexFunc(locations, func(item Location) bool {
		return item.HID == hid
	})
	if index < 0 {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, errorResponse{Code: "location_not_found", Message: "location not found"})
		return
	}

	locations = append(locations[:index], locations[index+1:]...)
	if err := api.storage.SaveLocations(locations); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{Code: "locations_write_failed", Message: err.Error()})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (api *LocationsAPI) handleCreateChildCategory(w http.ResponseWriter, r *http.Request) {
	categoryKey := strings.TrimSpace(chi.URLParam(r, "category"))
	if categoryKey == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse{Code: "invalid_category", Message: "category is required"})
		return
	}

	var request addChildCategoryRequest
	if err := render.DecodeJSON(r.Body, &request); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse{Code: "invalid_body", Message: "invalid child category body"})
		return
	}

	config, err := api.storage.LoadCategoryConfig()
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{Code: "config_read_failed", Message: err.Error()})
		return
	}

	label := strings.TrimSpace(request.Label)
	if label == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse{Code: "validation_failed", Message: "label is required", Fields: map[string]string{"label": "label is required"}})
		return
	}

	key := strings.TrimSpace(request.Key)
	if key == "" {
		key = slugifyKey(label)
	}

	for categoryIndex := range config.Categories {
		category := &config.Categories[categoryIndex]
		if category.Key != categoryKey {
			continue
		}

		for _, child := range category.Children {
			if child.Key == key {
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, errorResponse{Code: "child_category_conflict", Message: "child category key already exists"})
				return
			}
		}

		created := CategoryChild{Key: key, Label: label}
		category.Children = append(category.Children, created)
		if err := api.storage.SaveCategoryConfig(config); err != nil {
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, errorResponse{Code: "config_write_failed", Message: err.Error()})
			return
		}

		render.Status(r, http.StatusCreated)
		render.JSON(w, r, created)
		return
	}

	render.Status(r, http.StatusNotFound)
	render.JSON(w, r, errorResponse{Code: "category_not_found", Message: "category not found"})
}

func (api *LocationsAPI) handleUpdateChildCategory(w http.ResponseWriter, r *http.Request) {
	categoryKey := strings.TrimSpace(chi.URLParam(r, "category"))
	childKey := strings.TrimSpace(chi.URLParam(r, "child"))
	if categoryKey == "" || childKey == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse{Code: "invalid_params", Message: "category and child are required"})
		return
	}

	var request updateChildCategoryRequest
	if err := render.DecodeJSON(r.Body, &request); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse{Code: "invalid_body", Message: "invalid child category body"})
		return
	}

	label := strings.TrimSpace(request.Label)
	if label == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse{Code: "validation_failed", Message: "label is required", Fields: map[string]string{"label": "label is required"}})
		return
	}

	newKey := strings.TrimSpace(request.Key)
	if newKey == "" {
		newKey = slugifyKey(label)
	}

	config, err := api.storage.LoadCategoryConfig()
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{Code: "config_read_failed", Message: err.Error()})
		return
	}

	locations, err := api.storage.LoadLocations()
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{Code: "locations_read_failed", Message: err.Error()})
		return
	}

	for categoryIndex := range config.Categories {
		category := &config.Categories[categoryIndex]
		if category.Key != categoryKey {
			continue
		}

		childIndex := slices.IndexFunc(category.Children, func(child CategoryChild) bool {
			return child.Key == childKey
		})
		if childIndex < 0 {
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, errorResponse{Code: "child_category_not_found", Message: "child category not found"})
			return
		}

		for index, child := range category.Children {
			if index == childIndex {
				continue
			}
			if child.Key == newKey {
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, errorResponse{Code: "child_category_conflict", Message: "child category key already exists"})
				return
			}
		}

		category.Children[childIndex] = CategoryChild{Key: newKey, Label: label}

		for index := range locations {
			if locations[index].Category != categoryKey {
				continue
			}

			if categoryKey == "bonetsky" {
				if locations[index].Type != childKey {
					continue
				}
				locations[index].Type = newKey
				locations[index].TypeDisplay = label
			} else {
				if locations[index].ChildCategory != childKey {
					continue
				}
				locations[index].ChildCategory = newKey
				locations[index].ChildCategoryDisplay = label
			}
		}

		if err := api.storage.SaveCategoryConfig(config); err != nil {
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, errorResponse{Code: "config_write_failed", Message: err.Error()})
			return
		}
		if err := api.storage.SaveLocations(locations); err != nil {
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, errorResponse{Code: "locations_write_failed", Message: err.Error()})
			return
		}

		render.JSON(w, r, category.Children[childIndex])
		return
	}

	render.Status(r, http.StatusNotFound)
	render.JSON(w, r, errorResponse{Code: "category_not_found", Message: "category not found"})
}

func (api *LocationsAPI) handleDeleteChildCategory(w http.ResponseWriter, r *http.Request) {
	categoryKey := strings.TrimSpace(chi.URLParam(r, "category"))
	childKey := strings.TrimSpace(chi.URLParam(r, "child"))
	if categoryKey == "" || childKey == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse{Code: "invalid_params", Message: "category and child are required"})
		return
	}

	config, err := api.storage.LoadCategoryConfig()
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{Code: "config_read_failed", Message: err.Error()})
		return
	}

	locations, err := api.storage.LoadLocations()
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{Code: "locations_read_failed", Message: err.Error()})
		return
	}

	for _, location := range locations {
		if location.Category != categoryKey {
			continue
		}

		if categoryKey == "bonetsky" {
			if location.Type == childKey {
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, errorResponse{Code: "child_category_in_use", Message: "child category is used by existing locations"})
				return
			}
			continue
		}

		if location.ChildCategory == childKey {
			render.Status(r, http.StatusConflict)
			render.JSON(w, r, errorResponse{Code: "child_category_in_use", Message: "child category is used by existing locations"})
			return
		}
	}

	for categoryIndex := range config.Categories {
		category := &config.Categories[categoryIndex]
		if category.Key != categoryKey {
			continue
		}

		childIndex := slices.IndexFunc(category.Children, func(child CategoryChild) bool {
			return child.Key == childKey
		})
		if childIndex < 0 {
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, errorResponse{Code: "child_category_not_found", Message: "child category not found"})
			return
		}

		category.Children = append(category.Children[:childIndex], category.Children[childIndex+1:]...)
		if err := api.storage.SaveCategoryConfig(config); err != nil {
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, errorResponse{Code: "config_write_failed", Message: err.Error()})
			return
		}

		w.WriteHeader(http.StatusNoContent)
		return
	}

	render.Status(r, http.StatusNotFound)
	render.JSON(w, r, errorResponse{Code: "category_not_found", Message: "category not found"})
}

func (api *LocationsAPI) loadLocationsAndConfig() ([]Location, CategoryConfigFile, error) {
	locations, err := api.storage.LoadLocations()
	if err != nil {
		return nil, CategoryConfigFile{}, err
	}

	config, err := api.storage.LoadCategoryConfig()
	if err != nil {
		return nil, CategoryConfigFile{}, err
	}

	return locations, config, nil
}

func (api *LocationsAPI) findLocationByHID(hid string) (Location, error) {
	locations, err := api.storage.LoadLocations()
	if err != nil {
		return Location{}, err
	}

	for _, location := range locations {
		if location.HID == hid {
			return location, nil
		}
	}

	return Location{}, errors.New("not found")
}

func hasLocationHID(locations []Location, hid string, skipHID string) bool {
	for _, location := range locations {
		if location.HID == hid && location.HID != skipHID {
			return true
		}
	}
	return false
}

func normalizeAndValidateLocation(location Location, config CategoryConfigFile) (Location, map[string]string) {
	normalized := location
	fieldErrors := map[string]string{}

	normalized.Name = strings.TrimSpace(location.Name)
	normalized.Address = strings.TrimSpace(location.Address)
	normalized.Manager = strings.TrimSpace(location.Manager)
	normalized.Category = strings.TrimSpace(location.Category)
	normalized.ChildCategory = strings.TrimSpace(location.ChildCategory)
	normalized.Type = strings.TrimSpace(location.Type)

	if normalized.Name == "" {
		fieldErrors["name"] = "name is required"
	}

	if normalized.Category == "" {
		fieldErrors["category"] = "category is required"
	}

	if normalized.Lat < -90 || normalized.Lat > 90 {
		fieldErrors["lat"] = "lat must be between -90 and 90"
	}

	if normalized.Lng < -180 || normalized.Lng > 180 {
		fieldErrors["lng"] = "lng must be between -180 and 180"
	}

	categoryConfig, childConfig, ok := findCategoryConfig(config, normalized.Category, normalized.ChildCategory)
	if !ok {
		fieldErrors["category"] = "category is not configured"
	}

	if ok {
		normalized.CategoryDisplay = categoryConfig.Label
		if normalized.Category == "bonetsky" {
			if normalized.ChildCategory == "" {
				fieldErrors["child_category"] = "type is required"
			} else if childConfig.Key == "" {
				fieldErrors["child_category"] = "type is not configured"
			} else {
				normalized.Type = normalized.ChildCategory
				normalized.TypeDisplay = childConfig.Label
				normalized.ChildCategory = normalized.Category
				normalized.ChildCategoryDisplay = categoryConfig.Label
			}
		} else {
			if normalized.ChildCategory == "" {
				fieldErrors["child_category"] = "child_category is required"
			} else if childConfig.Key == "" {
				fieldErrors["child_category"] = "child_category is not configured"
			} else {
				normalized.ChildCategory = childConfig.Key
				normalized.ChildCategoryDisplay = childConfig.Label
				normalized.Type = ""
				normalized.TypeDisplay = ""
			}
		}
	}

	return normalized, fieldErrors
}

func findCategoryConfig(config CategoryConfigFile, categoryKey string, childKey string) (CategoryConfig, CategoryChild, bool) {
	for _, category := range config.Categories {
		if category.Key != categoryKey {
			continue
		}

		if categoryKey == "bonetsky" {
			for _, child := range category.Children {
				if child.Key == childKey {
					return category, child, true
				}
			}
			return category, CategoryChild{}, true
		}

		for _, child := range category.Children {
			if child.Key == childKey {
				return category, child, true
			}
		}
		return category, CategoryChild{}, true
	}

	return CategoryConfig{}, CategoryChild{}, false
}
