package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestLocationsCRUDFlow(t *testing.T) {
	dataDir := t.TempDir()
	writeTestJSON(t, filepath.Join(dataDir, "locations.json"), locationsFile{Locations: []Location{}})
	writeTestJSON(t, filepath.Join(dataDir, "location_categories.json"), CategoryConfigFile{
		Categories: []CategoryConfig{
			{
				Key:   "chastnyi",
				Label: "Частные клиники",
				Children: []CategoryChild{
					{Key: "ginekologiia", Label: "Гинекология"},
				},
			},
		},
	})

	router := chi.NewRouter()
	RegisterRoute(router, dataDir, "secret-key", func(name string) ([]byte, error) {
		return os.ReadFile(filepath.Join(dataDir, name))
	}, false)

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

	var created Location
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

	var updated Location
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
	router := chi.NewRouter()
	locAPI := RegisterRoute(router, dataDir, "secret-key", func(name string) ([]byte, error) {
		return os.ReadFile(filepath.Join(dataDir, name))
	}, true)

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
	writeTestJSON(t, filepath.Join(dataDir, "locations.json"), locationsFile{Locations: []Location{}})
	writeTestJSON(t, filepath.Join(dataDir, "location_categories.json"), CategoryConfigFile{
		Categories: []CategoryConfig{
			{
				Key:   "chastnyi",
				Label: "Частные клиники",
				Children: []CategoryChild{
					{Key: "ginekologiia", Label: "Гинекология"},
				},
			},
		},
	})

	router := chi.NewRouter()
	RegisterRoute(router, dataDir, "secret-key", func(name string) ([]byte, error) {
		return os.ReadFile(filepath.Join(dataDir, name))
	}, false)

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

	configPayload, err := os.ReadFile(filepath.Join(dataDir, "location_categories.json"))
	if err != nil {
		t.Fatalf("cannot read config file: %v", err)
	}

	var config CategoryConfigFile
	if err := json.Unmarshal(configPayload, &config); err != nil {
		t.Fatalf("cannot decode config: %v", err)
	}

	if len(config.Categories[0].Children) != 2 {
		t.Fatalf("expected 2 child categories, got %d", len(config.Categories[0].Children))
	}
}

func TestUpdateChildCategory(t *testing.T) {
	dataDir := t.TempDir()
	writeTestJSON(t, filepath.Join(dataDir, "locations.json"), locationsFile{
		Locations: []Location{
			{
				HID:                  "1",
				Name:                 "Клиника",
				Category:             "chastnyi",
				ChildCategory:        "ginekologiia",
				ChildCategoryDisplay: "Гинекология",
				CategoryDisplay:      "Частные клиники",
				Lat:                  42.8,
				Lng:                  74.6,
			},
		},
	})
	writeTestJSON(t, filepath.Join(dataDir, "location_categories.json"), CategoryConfigFile{
		Categories: []CategoryConfig{
			{
				Key:   "chastnyi",
				Label: "Частные клиники",
				Children: []CategoryChild{
					{Key: "ginekologiia", Label: "Гинекология"},
				},
			},
		},
	})

	router := chi.NewRouter()
	RegisterRoute(router, dataDir, "secret-key", func(name string) ([]byte, error) {
		return os.ReadFile(filepath.Join(dataDir, name))
	}, false)
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

	payload, err := os.ReadFile(filepath.Join(dataDir, "locations.json"))
	if err != nil {
		t.Fatalf("cannot read locations file: %v", err)
	}
	var wrapped locationsFile
	if err := json.Unmarshal(payload, &wrapped); err != nil {
		t.Fatalf("cannot decode locations file: %v", err)
	}
	if wrapped.Locations[0].ChildCategory != "endokrinologiia" {
		t.Fatalf("expected updated child category key, got %q", wrapped.Locations[0].ChildCategory)
	}
}

func TestDeleteChildCategoryInUseFails(t *testing.T) {
	dataDir := t.TempDir()
	writeTestJSON(t, filepath.Join(dataDir, "locations.json"), locationsFile{
		Locations: []Location{
			{
				HID:           "1",
				Name:          "Клиника",
				Category:      "chastnyi",
				ChildCategory: "ginekologiia",
				Lat:           42.8,
				Lng:           74.6,
			},
		},
	})
	writeTestJSON(t, filepath.Join(dataDir, "location_categories.json"), CategoryConfigFile{
		Categories: []CategoryConfig{
			{
				Key:   "chastnyi",
				Label: "Частные клиники",
				Children: []CategoryChild{
					{Key: "ginekologiia", Label: "Гинекология"},
				},
			},
		},
	})

	router := chi.NewRouter()
	RegisterRoute(router, dataDir, "secret-key", func(name string) ([]byte, error) {
		return os.ReadFile(filepath.Join(dataDir, name))
	}, false)
	token := issueTestToken(t, router, "/api/location-config/chastnyi/children/ginekologiia", http.MethodDelete)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodDelete, "/api/location-config/chastnyi/children/ginekologiia", nil)
	request.Header.Set(tokenHeader, token)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", recorder.Code, recorder.Body.String())
	}
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

func writeTestJSON(t *testing.T, path string, value any) {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("cannot marshal test json: %v", err)
	}
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatalf("cannot write test json: %v", err)
	}
}
