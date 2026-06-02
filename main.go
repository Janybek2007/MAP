package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	dataDir            = "data"
	serverMarkerHeader = "X-Map-Server"
	defaultPort        = 27436
)

var runMode = "prod"
var currentServer *http.Server

func main() {
	mode := "prod"
	port := defaultPort
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
	runMode = mode

	mux := http.NewServeMux()

	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/shutdown", shutdownHandler)
	mux.HandleFunc("/data/", dataFileHandler)
	mux.HandleFunc("/data", dataIndexHandler)
	if mode == "prod" {
		if err := registerStaticRoutes(mux); err != nil {
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
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

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

func dataIndexHandler(w http.ResponseWriter, _ *http.Request) {
	var entries []fs.DirEntry
	if runMode == "dev" {
		diskEntries, err := os.ReadDir(dataDir)
		if err != nil {
			http.Error(w, "cannot read data dir", http.StatusInternalServerError)
			return
		}
		entries = make([]fs.DirEntry, 0, len(diskEntries))
		for _, entry := range diskEntries {
			entries = append(entries, entry)
		}
	} else {
		dataFS, err := fs.Sub(appFS, dataDir)
		if err != nil {
			http.Error(w, "cannot access embedded data dir", http.StatusInternalServerError)
			return
		}
		embeddedEntries, err := fs.ReadDir(dataFS, ".")
		if err != nil {
			http.Error(w, "cannot read data dir", http.StatusInternalServerError)
			return
		}
		entries = embeddedEntries
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(strings.ToLower(name), ".json") {
			files = append(files, name)
		}
	}
	sort.Strings(files)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"files": files,
	})
}

func dataFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	urlPath := strings.TrimPrefix(r.URL.Path, "/data/")
	urlPath = strings.Trim(urlPath, "/")
	if urlPath == "" {
		http.NotFound(w, r)
		return
	}

	parts := strings.Split(urlPath, "/")
	if len(parts) == 3 && parts[2] == "coords" {
		handleCoordsByHID(w, parts[0], parts[1])
		return
	}

	name := filepath.Base(parts[0])
	if name == "." || name == "" {
		http.NotFound(w, r)
		return
	}
	if !strings.HasSuffix(strings.ToLower(name), ".json") {
		http.Error(w, "only .json files are allowed", http.StatusBadRequest)
		return
	}

	var payload []byte
	var err error
	if runMode == "dev" {
		fullPath := filepath.Join(dataDir, name)
		payload, err = os.ReadFile(fullPath)
	} else {
		payload, err = appFS.ReadFile(path.Join(dataDir, name))
	}
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var data any
	if err := json.Unmarshal(payload, &data); err != nil {
		http.Error(w, "invalid json in source file", http.StatusInternalServerError)
		return
	}

	if shouldStripCoords(name) {
		data = stripCoords(data)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(data)
}

func shouldStripCoords(name string) bool {
	lower := strings.ToLower(name)
	return lower == "districts.json" || lower == "cities.json" || lower == "regions.json"
}

func stripCoords(data any) any {
	switch typed := data.(type) {
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			if obj, ok := item.(map[string]any); ok {
				copyObj := map[string]any{}
				for key, val := range obj {
					if key == "coords" {
						continue
					}
					copyObj[key] = val
				}
				out = append(out, copyObj)
			} else {
				out = append(out, item)
			}
		}
		return out
	default:
		return data
	}
}

func handleCoordsByHID(w http.ResponseWriter, collection string, hid string) {
	if hid == "" {
		http.Error(w, "hid is required", http.StatusBadRequest)
		return
	}

	fileName := strings.ToLower(collection) + ".json"
	if fileName != "districts.json" && fileName != "cities.json" && fileName != "regions.json" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	var payload []byte
	var err error
	if runMode == "dev" {
		payload, err = os.ReadFile(filepath.Join(dataDir, fileName))
	} else {
		payload, err = appFS.ReadFile(path.Join(dataDir, fileName))
	}
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	var data []map[string]any
	if err := json.Unmarshal(payload, &data); err != nil {
		http.Error(w, "invalid json in source file", http.StatusInternalServerError)
		return
	}

	for _, item := range data {
		itemHID, _ := item["hid"].(string)
		if itemHID != hid {
			continue
		}
		coords, ok := item["coords"]
		if !ok {
			coords = []any{}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(coords)
		return
	}

	http.Error(w, "not found", http.StatusNotFound)
}

func registerStaticRoutes(mux *http.ServeMux) error {
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
			if serveEmbeddedFile(w, r, webFS, filePath) {
				return
			}
			http.NotFound(w, r)
			return
		}

		_ = serveEmbeddedFile(w, r, webFS, "index.html")
	})

	return nil
}

func hasExtension(filePath string) bool {
	base := path.Base(filePath)
	return strings.Contains(base, ".")
}

func serveEmbeddedFile(w http.ResponseWriter, r *http.Request, files fs.FS, filePath string) bool {
	file, err := files.Open(filePath)
	if err != nil {
		return false
	}
	_ = file.Close()
	http.ServeFileFS(w, r, files, filePath)
	return true
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
