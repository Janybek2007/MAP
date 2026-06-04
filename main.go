package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"map/api"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
)

const (
	dataDir            = "data"
	serverMarkerHeader = "X-Map-Server"
	defaultProdPort    = 27436
	defaultDevPort     = 8080
)

var runMode = "prod"
var currentServer *http.Server

func main() {
	mode := "prod"
	port := defaultProdPort
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "-mode=") {
			mode = strings.TrimPrefix(arg, "-mode=")
		} else if strings.HasPrefix(arg, "-port=") {
			if n, err := strconv.Atoi(strings.TrimPrefix(arg, "-port=")); err == nil && n > 0 && n < 65536 {
				port = n
			}
		}
	}
	if mode != "dev" && mode != "prod" {
		log.Fatalf("invalid -mode=%s (use dev|prod)", mode)
	}
	if port == defaultProdPort && mode == "dev" {
		port = defaultDevPort
	}
	runMode = mode
	loadEnvFiles(mode)

	mux := http.NewServeMux()
	apiRouter := chi.NewRouter()

	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/shutdown", shutdownHandler)
	apiKey := strings.TrimSpace(os.Getenv("API_KEY"))
	if apiKey == "" {
		log.Fatal("API_KEY is required")
	}
	locAPI := api.RegisterRoute(apiRouter, dataDir, apiKey, readDataFile, mode == "prod")
	mux.Handle("/api/", apiRouter)
	dataRouter := locAPI.NewDataRouter()
	mux.Handle("/data/", dataRouter)
	if mode == "prod" {
		if err := registerStaticRoutes(mux, locAPI); err != nil {
			log.Fatal(err)
		}
	}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: withCommonHeaders(mux),
	}
	currentServer = server

	listener, finalPort, reused, err := listenOrFind(port, mode == "prod")
	if err != nil {
		log.Fatal(err)
	}
	if reused {
		// Our server already runs on that port; ask it to shutdown and take over.
		if mode == "prod" {
			_ = shutdownOurServer(finalPort)
			listener, finalPort, _, err = listenOrFind(port, false)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			_ = shutdownOurServer(finalPort)
			listener, finalPort, _, err = listenOrFind(port, false)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	server.Addr = fmt.Sprintf(":%d", finalPort)

	log.Printf("server started on :%d (mode=%s)", finalPort, mode)
	log.Printf("data dir: %s (embedded)", dataDir)

	if mode == "prod" {
		go func(p int) {
			healthURL := fmt.Sprintf("http://localhost:%d/health", p)
			_ = waitForServerReady(healthURL, 6*time.Second)
			_ = waitForHTTP200(fmt.Sprintf("http://localhost:%d/", p), 6*time.Second)
			_ = openBrowser(fmt.Sprintf("http://localhost:%d", p))
		}(finalPort)
	}

	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func shutdownHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.Header.Get(serverMarkerHeader) != "1" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	host := r.Host
	if host != "localhost" && !strings.HasPrefix(host, "localhost:") && !strings.HasPrefix(host, "127.0.0.1:") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	w.WriteHeader(http.StatusNoContent)
	if currentServer == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = currentServer.Shutdown(ctx)
	}()
}

func shutdownOurServer(port int) error {
	client := &http.Client{Timeout: 600 * time.Millisecond}
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:%d/shutdown", port), nil)
	if err != nil {
		return err
	}
	req.Header.Set(serverMarkerHeader, "1")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	// wait until port is free
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			_ = ln.Close()
			return nil
		}
		time.Sleep(120 * time.Millisecond)
	}
	return nil
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func withCommonHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Resource-Token, X-Token")
		w.Header().Set("Access-Control-Expose-Headers", "X-Next-Token, X-Next-Token-Expires-At")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set(serverMarkerHeader, "1")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func registerStaticRoutes(mux *http.ServeMux, locAPI *api.LocationsAPI) error {
	webFS, err := fs.Sub(appFS, "build")
	if err != nil {
		return err
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		cleanPath := path.Clean("/" + strings.TrimPrefix(r.URL.Path, "/"))
		filePath := strings.TrimPrefix(cleanPath, "/")
		if filePath == "" {
			filePath = "index.html"
		}

		if hasExtension(filePath) {
			if serveEmbeddedFile(w, r, webFS, filePath, locAPI) {
				return
			}
			http.NotFound(w, r)
			return
		}

		if serveEmbeddedFile(w, r, webFS, filePath+".html", locAPI) {
			return
		}

		if serveEmbeddedFile(w, r, webFS, path.Join(filePath, "index.html"), locAPI) {
			return
		}

		_ = serveEmbeddedFile(w, r, webFS, "index.html", locAPI)
	})

	return nil
}

func hasExtension(filePath string) bool {
	base := path.Base(filePath)
	return strings.Contains(base, ".")
}

func serveEmbeddedFile(w http.ResponseWriter, r *http.Request, files fs.FS, filePath string, locAPI *api.LocationsAPI) bool {
	file, err := files.Open(filePath)
	if err != nil {
		return false
	}
	_ = file.Close()

	if strings.HasSuffix(strings.ToLower(filePath), ".html") {
		payload, err := fs.ReadFile(files, filePath)
		if err != nil {
			http.Error(w, "failed to read page", http.StatusInternalServerError)
			return true
		}

		payload, err = injectTokenEndpointToken(payload, locAPI)
		if err != nil {
			http.Error(w, "failed to prepare page", http.StatusInternalServerError)
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

func readDataFile(name string) ([]byte, error) {
	fullPath := filepath.Join(dataDir, name)
	if payload, err := os.ReadFile(fullPath); err == nil {
		return payload, nil
	}

	if runMode == "prod" {
		return appFS.ReadFile(path.Join(dataDir, name))
	}

	return os.ReadFile(fullPath)
}

func loadEnvFiles(mode string) {
	_ = godotenv.Load(".env")
	if mode == "dev" {
		_ = godotenv.Overload(".env.development")
		return
	}
	_ = godotenv.Overload(".env.production")
}

func listenOrFind(startPort int, allowReuse bool) (net.Listener, int, bool, error) {
	const maxAttempts = 16
	for attempt := 0; attempt < maxAttempts; attempt++ {
		p := startPort + attempt
		addr := fmt.Sprintf(":%d", p)
		ln, err := net.Listen("tcp", addr)
		if err == nil {
			return ln, p, false, nil
		}

		if allowReuse && isOurServer(p) {
			return nil, p, true, nil
		}
	}
	return nil, startPort, false, fmt.Errorf("no free port found starting from %d", startPort)
}

func isOurServer(port int) bool {
	client := &http.Client{Timeout: 350 * time.Millisecond}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/health", port))
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}
	if resp.Header.Get(serverMarkerHeader) != "1" {
		return false
	}
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return false
	}
	return payload["status"] == "ok"
}

func waitForServerReady(url string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 350 * time.Millisecond}
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return true
			}
		}
		time.Sleep(120 * time.Millisecond)
	}
	return false
}

func waitForHTTP200(url string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 500 * time.Millisecond}
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return true
			}
		}
		time.Sleep(120 * time.Millisecond)
	}
	return false
}
