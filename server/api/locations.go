package api

import (
	"encoding/json"
	"errors"
	"map/model"
	"map/repository"
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
	repository           repository.Repository
	tokenManager         *TokenManager
	protectTokenEndpoint bool
}

func NewLocationsAPI(router chi.Router, repo repository.Repository, tokenManager *TokenManager, protectTokenEndpoint bool) *LocationsAPI {
	return &LocationsAPI{
		router:               router,
		repository:           repo,
		tokenManager:         tokenManager,
		protectTokenEndpoint: protectTokenEndpoint,
	}
}

func RegisterRoute(router chi.Router, apiKey string, repo repository.Repository, protectTokenEndpoint bool) *LocationsAPI {
	api := NewLocationsAPI(router, repo, NewTokenManager(apiKey), protectTokenEndpoint)
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
			render.JSON(w, r, errorResponse{Code: "missing_token", Message: "токен доступа к ресурсу обязателен"})
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
			render.JSON(w, r, errorResponse{Code: "missing_token_endpoint_token", Message: "токен для маршрута выдачи токенов обязателен"})
			return
		}

		if err := api.tokenManager.ValidateTokenEndpoint(token); err != nil {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, errorResponse{Code: "invalid_token_endpoint_token", Message: err.Error()})
			return
		}

		if !api.setNextTokenEndpointHeaders(w) {
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, errorResponse{Code: "token_endpoint_issue_failed", Message: "не удалось выдать следующий токен для маршрута выдачи токенов"})
			return
		}
	}

	var raw json.RawMessage
	if err := render.DecodeJSON(r.Body, &raw); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse{Code: "invalid_body", Message: "неверное тело запроса на выдачу токена"})
		return
	}

	if len(raw) > 0 && raw[0] == '[' {
		var requests []tokenRequest
		if err := json.Unmarshal(raw, &requests); err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, errorResponse{Code: "invalid_body", Message: "неверное тело запроса на выдачу токена"})
			return
		}
		responses := make([]tokenResponse, len(requests))
		for index, request := range requests {
			response, err := api.tokenManager.Issue(request.Method, request.URL)
			if err != nil {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, errorResponse{Code: "invalid_token_request", Message: err.Error()})
				return
			}
			responses[index] = response
		}
		render.JSON(w, r, responses)
		return
	}

	var request tokenRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse{Code: "invalid_body", Message: "неверное тело запроса на выдачу токена"})
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
	locations, err := api.repository.ListLocations()
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
		render.JSON(w, r, errorResponse{Code: "location_not_found", Message: "локация не найдена"})
		return
	}

	render.JSON(w, r, location)
}

func (api *LocationsAPI) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	config, err := api.repository.LoadCategoryConfig()
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{Code: "config_read_failed", Message: err.Error()})
		return
	}

	render.JSON(w, r, config)
}

func (api *LocationsAPI) handleCreateLocation(w http.ResponseWriter, r *http.Request) {
	var location model.Location
	if err := render.DecodeJSON(r.Body, &location); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse{Code: "invalid_body", Message: "неверное тело запроса локации"})
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
		render.JSON(w, r, errorResponse{Code: "validation_failed", Message: "не удалось пройти валидацию локации", Fields: fieldErrors})
		return
	}

	normalized.HID = repository.GenerateLocationHID(normalized.Name)
	if hasLocationHID(locations, normalized.HID, "") {
		render.Status(r, http.StatusConflict)
		render.JSON(w, r, errorResponse{Code: "hid_conflict", Message: "локация с таким сгенерированным идентификатором уже существует"})
		return
	}

	if err := api.repository.CreateLocation(normalized); err != nil {
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
		render.JSON(w, r, errorResponse{Code: "invalid_hid", Message: "идентификатор обязателен"})
		return
	}

	var location model.Location
	if err := render.DecodeJSON(r.Body, &location); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse{Code: "invalid_body", Message: "неверное тело запроса локации"})
		return
	}

	locations, config, err := api.loadLocationsAndConfig()
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{Code: "storage_error", Message: err.Error()})
		return
	}

	index := slices.IndexFunc(locations, func(item model.Location) bool {
		return item.HID == hid
	})
	if index < 0 {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, errorResponse{Code: "location_not_found", Message: "локация не найдена"})
		return
	}

	normalized, fieldErrors := normalizeAndValidateLocation(location, config)
	if len(fieldErrors) > 0 {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse{Code: "validation_failed", Message: "не удалось пройти валидацию локации", Fields: fieldErrors})
		return
	}

	normalized.HID = repository.GenerateLocationHID(normalized.Name)
	if hasLocationHID(locations, normalized.HID, hid) {
		render.Status(r, http.StatusConflict)
		render.JSON(w, r, errorResponse{Code: "hid_conflict", Message: "локация с таким сгенерированным идентификатором уже существует"})
		return
	}

	if err := api.repository.UpdateLocation(hid, normalized); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, errorResponse{Code: "location_not_found", Message: "локация не найдена"})
			return
		}
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
		render.JSON(w, r, errorResponse{Code: "invalid_hid", Message: "идентификатор обязателен"})
		return
	}

	if err := api.repository.DeleteLocation(hid); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, errorResponse{Code: "location_not_found", Message: "локация не найдена"})
			return
		}
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
		render.JSON(w, r, errorResponse{Code: "invalid_category", Message: "категория обязательна"})
		return
	}

	var request addChildCategoryRequest
	if err := render.DecodeJSON(r.Body, &request); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse{Code: "invalid_body", Message: "неверное тело запроса подкатегории"})
		return
	}

	config, err := api.repository.LoadCategoryConfig()
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{Code: "config_read_failed", Message: err.Error()})
		return
	}

	label := strings.TrimSpace(request.Label)
	if label == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse{Code: "validation_failed", Message: "название обязательно", Fields: map[string]string{"label": "название обязательно"}})
		return
	}

	key := strings.TrimSpace(request.Key)
	if key == "" {
		key = repository.SlugifyKey(label)
	}

	for _, category := range config.Categories {
		if category.Key != categoryKey {
			continue
		}

		created, err := api.repository.CreateChildCategory(categoryKey, model.CategoryChild{Key: key, Label: label})
		if errors.Is(err, repository.ErrChildCategoryExists) {
			render.Status(r, http.StatusConflict)
			render.JSON(w, r, errorResponse{Code: "child_category_conflict", Message: "ключ подкатегории уже существует"})
			return
		}
		if err != nil {
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, errorResponse{Code: "config_write_failed", Message: err.Error()})
			return
		}

		render.Status(r, http.StatusCreated)
		render.JSON(w, r, created)
		return
	}

	render.Status(r, http.StatusNotFound)
	render.JSON(w, r, errorResponse{Code: "category_not_found", Message: "категория не найдена"})
}

func (api *LocationsAPI) handleUpdateChildCategory(w http.ResponseWriter, r *http.Request) {
	categoryKey := strings.TrimSpace(chi.URLParam(r, "category"))
	childKey := strings.TrimSpace(chi.URLParam(r, "child"))
	if categoryKey == "" || childKey == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse{Code: "invalid_params", Message: "категория и подкатегория обязательны"})
		return
	}

	var request updateChildCategoryRequest
	if err := render.DecodeJSON(r.Body, &request); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse{Code: "invalid_body", Message: "неверное тело запроса подкатегории"})
		return
	}

	label := strings.TrimSpace(request.Label)
	if label == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse{Code: "validation_failed", Message: "название обязательно", Fields: map[string]string{"label": "название обязательно"}})
		return
	}

	newKey := strings.TrimSpace(request.Key)
	if newKey == "" {
		newKey = repository.SlugifyKey(label)
	}

	config, err := api.repository.LoadCategoryConfig()
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{Code: "config_read_failed", Message: err.Error()})
		return
	}

	for _, category := range config.Categories {
		if category.Key != categoryKey {
			continue
		}

		updated, err := api.repository.UpdateChildCategory(categoryKey, childKey, model.CategoryChild{Key: newKey, Label: label})
		if errors.Is(err, repository.ErrChildCategoryExists) {
			render.Status(r, http.StatusConflict)
			render.JSON(w, r, errorResponse{Code: "child_category_conflict", Message: "ключ подкатегории уже существует"})
			return
		}
		if errors.Is(err, repository.ErrNotFound) {
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, errorResponse{Code: "child_category_not_found", Message: "подкатегория не найдена"})
			return
		}
		if err != nil {
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, errorResponse{Code: "config_write_failed", Message: err.Error()})
			return
		}

		render.JSON(w, r, updated)
		return
	}

	render.Status(r, http.StatusNotFound)
	render.JSON(w, r, errorResponse{Code: "category_not_found", Message: "категория не найдена"})
}

func (api *LocationsAPI) handleDeleteChildCategory(w http.ResponseWriter, r *http.Request) {
	categoryKey := strings.TrimSpace(chi.URLParam(r, "category"))
	childKey := strings.TrimSpace(chi.URLParam(r, "child"))
	if categoryKey == "" || childKey == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse{Code: "invalid_params", Message: "категория и подкатегория обязательны"})
		return
	}

	config, err := api.repository.LoadCategoryConfig()
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{Code: "config_read_failed", Message: err.Error()})
		return
	}

	for _, category := range config.Categories {
		if category.Key != categoryKey {
			continue
		}

		err := api.repository.DeleteChildCategory(categoryKey, childKey)
		if errors.Is(err, repository.ErrChildCategoryInUse) {
			render.Status(r, http.StatusConflict)
			render.JSON(w, r, errorResponse{Code: "child_category_in_use", Message: "подкатегория используется в существующих локациях"})
			return
		}
		if errors.Is(err, repository.ErrNotFound) {
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, errorResponse{Code: "child_category_not_found", Message: "подкатегория не найдена"})
			return
		}
		if err != nil {
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, errorResponse{Code: "config_write_failed", Message: err.Error()})
			return
		}

		w.WriteHeader(http.StatusNoContent)
		return
	}

	render.Status(r, http.StatusNotFound)
	render.JSON(w, r, errorResponse{Code: "category_not_found", Message: "категория не найдена"})
}

func (api *LocationsAPI) loadLocationsAndConfig() ([]model.Location, model.CategoryConfigFile, error) {
	locations, err := api.repository.ListLocations()
	if err != nil {
		return nil, model.CategoryConfigFile{}, err
	}

	config, err := api.repository.LoadCategoryConfig()
	if err != nil {
		return nil, model.CategoryConfigFile{}, err
	}

	return locations, config, nil
}

func (api *LocationsAPI) findLocationByHID(hid string) (model.Location, error) {
	location, err := api.repository.GetLocationByHID(hid)
	if err != nil {
		return model.Location{}, err
	}

	return location, nil
}

func hasLocationHID(locations []model.Location, hid string, skipHID string) bool {
	for _, location := range locations {
		if location.HID == hid && location.HID != skipHID {
			return true
		}
	}
	return false
}

func normalizeAndValidateLocation(location model.Location, config model.CategoryConfigFile) (model.Location, map[string]string) {
	normalized := location
	fieldErrors := map[string]string{}

	normalized.Name = strings.TrimSpace(location.Name)
	normalized.Address = strings.TrimSpace(location.Address)
	normalized.Manager = strings.TrimSpace(location.Manager)
	normalized.Category = strings.TrimSpace(location.Category)
	normalized.ChildCategory = strings.TrimSpace(location.ChildCategory)
	normalized.Type = strings.TrimSpace(location.Type)

	if normalized.Name == "" {
		fieldErrors["name"] = "название обязательно"
	}

	if normalized.Category == "" {
		fieldErrors["category"] = "категория обязательна"
	}

	if normalized.Lat < -90 || normalized.Lat > 90 {
		fieldErrors["lat"] = "широта должна быть в диапазоне от -90 до 90"
	}

	if normalized.Lng < -180 || normalized.Lng > 180 {
		fieldErrors["lng"] = "долгота должна быть в диапазоне от -180 до 180"
	}

	categoryConfig, childConfig, ok := findCategoryConfig(config, normalized.Category, normalized.ChildCategory)
	if !ok {
		fieldErrors["category"] = "категория не настроена"
	}

	if ok {
		normalized.CategoryDisplay = categoryConfig.Label
		if normalized.Category == "bonetsky" {
			if normalized.ChildCategory == "" {
				fieldErrors["child_category"] = "тип обязателен"
			} else if childConfig.Key == "" {
				fieldErrors["child_category"] = "тип не настроен"
			} else {
				normalized.Type = normalized.ChildCategory
				normalized.TypeDisplay = childConfig.Label
				normalized.ChildCategory = normalized.Category
				normalized.ChildCategoryDisplay = categoryConfig.Label
			}
		} else {
			if normalized.ChildCategory == "" {
				fieldErrors["child_category"] = "подкатегория обязательна"
			} else if childConfig.Key == "" {
				fieldErrors["child_category"] = "подкатегория не настроена"
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

func findCategoryConfig(config model.CategoryConfigFile, categoryKey string, childKey string) (model.CategoryConfig, model.CategoryChild, bool) {
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
			return category, model.CategoryChild{}, true
		}

		for _, child := range category.Children {
			if child.Key == childKey {
				return category, child, true
			}
		}
		return category, model.CategoryChild{}, true
	}

	return model.CategoryConfig{}, model.CategoryChild{}, false
}
