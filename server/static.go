package main

import (
	"encoding/json"
	"io/fs"
	"map/api"
	"net/http"
	"path"
	"strings"
)

func (application *App) registerStaticRoutes(mux *http.ServeMux, locAPI *api.LocationsAPI) error {
	webFS, err := fs.Sub(application.appFS, "build")
	if err != nil {
		return err
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "метод не поддерживается", http.StatusMethodNotAllowed)
			return
		}

		cleanPath := path.Clean("/" + strings.TrimPrefix(r.URL.Path, "/"))
		filePath := strings.TrimPrefix(cleanPath, "/")
		if filePath == "" {
			filePath = "index.html"
		}

		if hasExtension(filePath) {
			if application.serveEmbeddedFile(w, r, webFS, filePath, locAPI) {
				return
			}
			http.NotFound(w, r)
			return
		}

		if application.serveEmbeddedFile(w, r, webFS, filePath+".html", locAPI) {
			return
		}

		if application.serveEmbeddedFile(w, r, webFS, path.Join(filePath, "index.html"), locAPI) {
			return
		}

		_ = application.serveEmbeddedFile(w, r, webFS, "index.html", locAPI)
	})

	return nil
}

func hasExtension(filePath string) bool {
	base := path.Base(filePath)
	return strings.Contains(base, ".")
}

func (application *App) serveEmbeddedFile(w http.ResponseWriter, r *http.Request, files fs.FS, filePath string, locAPI *api.LocationsAPI) bool {
	file, err := files.Open(filePath)
	if err != nil {
		return false
	}
	_ = file.Close()

	if strings.HasSuffix(strings.ToLower(filePath), ".html") {
		payload, err := fs.ReadFile(files, filePath)
		if err != nil {
			http.Error(w, "не удалось прочитать страницу", http.StatusInternalServerError)
			return true
		}

		payload, err = injectTokenEndpointToken(payload, locAPI)
		if err != nil {
			http.Error(w, "не удалось подготовить страницу", http.StatusInternalServerError)
			return true
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
		return true
	}

	http.ServeFileFS(w, r, files, filePath)
	return true
}

func injectTokenEndpointToken(payload []byte, locAPI *api.LocationsAPI) ([]byte, error) {
	token, expiresAt, err := locAPI.IssueTokenEndpointToken()
	if err != nil {
		return nil, err
	}

	tokenPayload, err := json.Marshal(map[string]any{
		"token":      token,
		"expires_at": expiresAt,
	})
	if err != nil {
		return nil, err
	}

	script := []byte("<script>window.TOKEN=" + string(tokenPayload) + ";</script>")
	lowerPayload := strings.ToLower(string(payload))
	headIndex := strings.Index(lowerPayload, "</head>")
	if headIndex < 0 {
		return append(script, payload...), nil
	}

	result := make([]byte, 0, len(payload)+len(script))
	result = append(result, payload[:headIndex]...)
	result = append(result, script...)
	result = append(result, payload[headIndex:]...)
	return result, nil
}
