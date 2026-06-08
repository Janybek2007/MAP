package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"map/model"
	"map/repository"
	dbsqlite "map/sqlite"

	"github.com/go-chi/chi/v5"
)

func TestLocationsCRUDFlow(t *testing.T) {
	dataDir := t.TempDir()
	database := setupSQLiteForTest(t, dataDir)
	defer database.Close()

	router := buildTestRouter(t, database, dataDir)

	token := issueTestToken(t, router, "/api/locations", http.MethodPost)

	createBody := map[string]any{
		"name":            "Тест Клиника",
		"address":         "Тестовый адрес",
		"category":        "chastnyi",
		"child_category":  "ginekologiia",
		"is_partnerships": true,
		"manager":         "Индира",
		"lat":             42.87,
		"lng":             74.6,
	}

	createRecorder := httptest.NewRecorder()
	createRequest := httptest.NewRequest(http.MethodPost, "/api/locations", mustJSONBody(t, createBody))
	createRequest.Header.Set("Content-Type", "application/json")
	createRequest.Header.Set(tokenHeader, token)
	router.ServeHTTP(createRecorder, createRequest)

	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", createRecorder.Code, createRecorder.Body.String())
	}

	var created model.Location
	if err := json.Unmarshal(createRecorder.Body.Bytes(), &created); err != nil {
		t.Fatalf("cannot decode created location: %v", err)
	}

	listRecorder := httptest.NewRecorder()
	listRequest := httptest.NewRequest(http.MethodGet, "/api/locations", nil)
	listRequest.Header.Set(tokenHeader, issueTestToken(t, router, "/api/locations", http.MethodGet))
	router.ServeHTTP(listRecorder, listRequest)
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", listRecorder.Code)
	}

	updateToken := issueTestToken(t, router, "/api/locations/"+created.HID, http.MethodPut)
	updateBody := map[string]any{
		"name":            "Тест Клиника 2",
		"address":         "Новый адрес",
		"category":        "chastnyi",
		"child_category":  "ginekologiia",
		"is_partnerships": false,
		"manager":         "Рахат",
		"lat":             42.88,
		"lng":             74.61,
	}

	updateRecorder := httptest.NewRecorder()
	updateRequest := httptest.NewRequest(http.MethodPut, "/api/locations/"+created.HID, mustJSONBody(t, updateBody))
	updateRequest.Header.Set("Content-Type", "application/json")
	updateRequest.Header.Set(tokenHeader, updateToken)
	router.ServeHTTP(updateRecorder, updateRequest)
	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", updateRecorder.Code, updateRecorder.Body.String())
	}

	var updated model.Location
	if err := json.Unmarshal(updateRecorder.Body.Bytes(), &updated); err != nil {
		t.Fatalf("cannot decode updated location: %v", err)
	}
	if updated.Name != "Тест Клиника 2" {
		t.Fatalf("expected updated name, got %q", updated.Name)
	}

	deleteToken := issueTestToken(t, router, "/api/locations/"+updated.HID, http.MethodDelete)
	deleteRecorder := httptest.NewRecorder()
	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/locations/"+updated.HID, nil)
	deleteRequest.Header.Set(tokenHeader, deleteToken)
	router.ServeHTTP(deleteRecorder, deleteRequest)
	if deleteRecorder.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", deleteRecorder.Code, deleteRecorder.Body.String())
	}
}

func TestTokenIsSingleUse(t *testing.T) {
	manager := NewTokenManager("secret-key")
	response, err := manager.Issue(http.MethodPost, "/api/locations")
	if err != nil {
		t.Fatalf("issue token failed: %v", err)
	}

	if err := manager.Validate(response.Token, http.MethodPost, "/api/locations"); err != nil {
		t.Fatalf("first token validation failed: %v", err)
	}

	if err := manager.Validate(response.Token, http.MethodPost, "/api/locations"); err == nil {
		t.Fatal("expected token reuse to fail")
	}
}

func TestTokenEndpointProtection(t *testing.T) {
	dataDir := t.TempDir()
	database := setupSQLiteForTest(t, dataDir)
	defer database.Close()

	router := chi.NewRouter()
	repo := repository.NewSQLiteRepository(database)
	locAPI := RegisterRoute(router, "secret-key", repo, true)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/tokens", mustJSONBody(t, map[string]string{
		"url":    "/api/locations",
		"method": http.MethodGet,
	}))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token endpoint token, got %d", recorder.Code)
	}

	bootstrapToken, _, err := locAPI.IssueTokenEndpointToken()
	if err != nil {
		t.Fatalf("issue bootstrap token failed: %v", err)
	}

	protectedRecorder := httptest.NewRecorder()
	protectedRequest := httptest.NewRequest(http.MethodPost, "/api/tokens", mustJSONBody(t, []map[string]string{
		{"url": "/api/locations", "method": http.MethodGet},
	}))
	protectedRequest.Header.Set("Content-Type", "application/json")
	protectedRequest.Header.Set(tokenEndpointHeader, bootstrapToken)
	router.ServeHTTP(protectedRecorder, protectedRequest)
	if protectedRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 with token endpoint token, got %d: %s", protectedRecorder.Code, protectedRecorder.Body.String())
	}
	if protectedRecorder.Header().Get(nextTokenHeader) == "" {
		t.Fatal("expected next token endpoint token header")
	}
}

func TestTokenEndpointTokenCannotBeUsedAsResourceToken(t *testing.T) {
	manager := NewTokenManager("secret-key")
	response, err := manager.IssueTokenEndpoint()
	if err != nil {
		t.Fatalf("issue bootstrap token failed: %v", err)
	}

	if err := manager.Validate(response.Token, http.MethodPost, "/api/tokens"); err == nil {
		t.Fatal("expected token endpoint token to fail as resource token")
	}
}

func TestAddChildCategory(t *testing.T) {
	dataDir := t.TempDir()
	database := setupSQLiteForTest(t, dataDir)
	defer database.Close()

	router := buildTestRouter(t, database, dataDir)
	token := issueTestToken(t, router, "/api/location-config/chastnyi/children", http.MethodPost)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/location-config/chastnyi/children",
		mustJSONBody(t, map[string]string{"label": "Эндокринология"}),
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(tokenHeader, token)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", recorder.Code, recorder.Body.String())
	}

	repo := repository.NewSQLiteRepository(database)
	config, err := repo.LoadCategoryConfig()
	if err != nil {
		t.Fatalf("cannot load category config: %v", err)
	}

	for _, category := range config.Categories {
		if category.Key != "chastnyi" {
			continue
		}
		if len(category.Children) != 3 {
			t.Fatalf("expected 3 child categories, got %d", len(category.Children))
		}
		return
	}
	t.Fatal("category chastnyi not found")
}

func TestUpdateChildCategory(t *testing.T) {
	dataDir := t.TempDir()
	database := setupSQLiteForTest(t, dataDir)
	defer database.Close()

	router := buildTestRouter(t, database, dataDir)
	token := issueTestToken(t, router, "/api/location-config/chastnyi/children/ginekologiia", http.MethodPut)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPut,
		"/api/location-config/chastnyi/children/ginekologiia",
		mustJSONBody(t, map[string]string{"label": "Эндокринология", "key": "endokrinologiia"}),
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(tokenHeader, token)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	repo := repository.NewSQLiteRepository(database)
	location, err := repo.GetLocationByHID("clinic-1")
	if err != nil {
		t.Fatalf("cannot get location: %v", err)
	}
	if location.ChildCategory != "endokrinologiia" {
		t.Fatalf("expected updated child category key, got %q", location.ChildCategory)
	}
}

func TestDeleteChildCategoryInUseFails(t *testing.T) {
	dataDir := t.TempDir()
	database := setupSQLiteForTest(t, dataDir)
	defer database.Close()

	router := buildTestRouter(t, database, dataDir)
	token := issueTestToken(t, router, "/api/location-config/chastnyi/children/ginekologiia", http.MethodDelete)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodDelete, "/api/location-config/chastnyi/children/ginekologiia", nil)
	request.Header.Set(tokenHeader, token)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func buildTestRouter(t *testing.T, database *sql.DB, dataDir string) http.Handler {
	t.Helper()

	router := chi.NewRouter()
	repo := repository.NewSQLiteRepository(database)
	RegisterRoute(router, "secret-key", repo, false)
	return router
}

func setupSQLiteForTest(t *testing.T, dataDir string) *sql.DB {
	t.Helper()

	writeBootstrapJSON(t, filepath.Join(dataDir, "location_categories.json"), model.CategoryConfigFile{
		Categories: []model.CategoryConfig{
			{Key: "bonetsky", Label: "Филиалы Бонецкого", Children: []model.CategoryChild{{Key: "pp", Label: "ПП"}}},
			{Key: "gos", Label: "ГОС", Children: []model.CategoryChild{{Key: "poliklinika", Label: "Поликлиника"}}},
			{Key: "rival", Label: "Конкуренты", Children: []model.CategoryChild{{Key: "rival_express", Label: "Экспресс"}}},
			{Key: "chastnyi", Label: "Частные клиники", Children: []model.CategoryChild{{Key: "ginekologiia", Label: "Гинекология"}, {Key: "pediatriia", Label: "Педиатрия"}}},
		},
	})

	writeBootstrapJSON(t, filepath.Join(dataDir, "locations.json"), model.LocationsFile{
		Locations: []model.Location{
			{
				HID:                  "clinic-1",
				Name:                 "Клиника",
				Address:              "Адрес",
				Category:             "chastnyi",
				ChildCategory:        "ginekologiia",
				CategoryDisplay:      "Частные клиники",
				ChildCategoryDisplay: "Гинекология",
				Lat:                  42.8,
				Lng:                  74.6,
			},
		},
	})

	writeBootstrapJSON(t, filepath.Join(dataDir, "cities.json"), []map[string]any{})
	writeBootstrapJSON(t, filepath.Join(dataDir, "districts.json"), []map[string]any{})
	writeBootstrapJSON(t, filepath.Join(dataDir, "regions.json"), []map[string]any{})

	databasePath := filepath.Join(dataDir, "test.sqlite")
	database, err := dbsqlite.OpenAndMigrate(databasePath)
	if err != nil {
		t.Fatalf("cannot open sqlite: %v", err)
	}

	if err := repository.BootstrapIfEmpty(database, func(name string) ([]byte, error) {
		return os.ReadFile(filepath.Join(dataDir, name))
	}); err != nil {
		t.Fatalf("cannot bootstrap sqlite: %v", err)
	}

	return database
}

func issueTestToken(t *testing.T, router http.Handler, url string, method string) string {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/tokens", mustJSONBody(t, map[string]string{
		"url":    url,
		"method": method,
	}))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected token 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var response tokenResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("cannot decode token response: %v", err)
	}

	return response.Token
}

func mustJSONBody(t *testing.T, value any) *bytes.Reader {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("cannot marshal json body: %v", err)
	}
	return bytes.NewReader(payload)
}

func writeBootstrapJSON(t *testing.T, path string, value any) {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("cannot marshal test json: %v", err)
	}
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatalf("cannot write test json: %v", err)
	}
}
