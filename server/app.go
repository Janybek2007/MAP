package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"map/api"
	"map/repository"
	dbsqlite "map/sqlite"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

type App struct {
	config        Config
	appFS         fs.FS
	database      *sql.DB
	repo          repository.Repository
	currentServer *http.Server
}

func Run(appFS fs.FS) error {
	config, err := loadConfig(osArgs())
	if err != nil {
		return err
	}

	database, err := dbsqlite.OpenExistingAndMigrate(config.SQLitePath)
	if err != nil {
		return err
	}

	repo := repository.NewSQLiteRepository(database)

	application := &App{
		config:   config,
		appFS:    appFS,
		database: database,
		repo:     repo,
	}

	return application.run()
}

func (application *App) run() error {
	defer application.database.Close()

	mux := http.NewServeMux()
	apiRouter := chi.NewRouter()

	mux.HandleFunc("/health", application.healthHandler)
	mux.HandleFunc("/shutdown", application.shutdownHandler)

	locAPI := api.RegisterRoute(
		apiRouter,
		application.config.APIKey,
		application.repo,
		application.config.ProtectTokenEndpoint,
	)

	mux.Handle("/api/", apiRouter)
	mux.Handle("/data/", locAPI.NewDataRouter())

	if application.config.Mode == "prod" {
		if err := application.registerStaticRoutes(mux, locAPI); err != nil {
			return err
		}
	}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", application.config.Port),
		Handler: withCommonHeaders(mux),
	}
	server.RegisterOnShutdown(func() {
		_ = application.database.Close()
	})
	application.currentServer = server

	listener, finalPort, reused, err := application.listenOrFind(application.config.Port, application.config.Mode == "prod")
	if err != nil {
		return err
	}

	if reused {
		if err := shutdownOurServer(finalPort); err != nil {
			return err
		}

		listener, finalPort, _, err = application.listenOrFind(application.config.Port, false)
		if err != nil {
			return err
		}
	}

	server.Addr = fmt.Sprintf(":%d", finalPort)

	log.Printf("сервер запущен на :%d (mode=%s)", finalPort, application.config.Mode)
	log.Printf("путь к sqlite: %s", application.config.SQLitePath)

	if application.config.Mode == "prod" {
		go func(port int) {
			healthURL := fmt.Sprintf("http://localhost:%d/health", port)
			_ = waitForServerReady(healthURL, 6*time.Second)
			_ = waitForHTTP200(fmt.Sprintf("http://localhost:%d/", port), 6*time.Second)
			_ = openBrowser(fmt.Sprintf("http://localhost:%d", port))
		}(finalPort)
	}

	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (application *App) healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set(serverMarkerHeader, "1")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (application *App) shutdownHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}
	if r.Header.Get(serverMarkerHeader) != "1" {
		http.Error(w, "доступ запрещён", http.StatusForbidden)
		return
	}
	host := r.Host
	if host != "localhost" && !strings.HasPrefix(host, "localhost:") && !strings.HasPrefix(host, "127.0.0.1:") {
		http.Error(w, "доступ запрещён", http.StatusForbidden)
		return
	}
	w.WriteHeader(http.StatusNoContent)
	if application.currentServer == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = application.currentServer.Shutdown(ctx)
	}()
}

func (application *App) listenOrFind(startPort int, allowReuse bool) (net.Listener, int, bool, error) {
	const maxAttempts = 16
	for attempt := 0; attempt < maxAttempts; attempt++ {
		port := startPort + attempt
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			return listener, port, false, nil
		}

		if allowReuse && isOurServer(port) {
			return nil, port, true, nil
		}
	}

	return nil, startPort, false, fmt.Errorf("не найден свободный порт, начиная с %d", startPort)
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
