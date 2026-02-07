package main

import (
	"cmp"
	"embed"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/handlers"
	"github.com/viscerous/goplaxt/lib/api"
	"github.com/viscerous/goplaxt/lib/store"
)

//go:embed static
var staticContent embed.FS

func main() {
	setupLogging()
	slog.Info("Starting Plaxt...")

	var storage store.Store
	if os.Getenv("POSTGRESQL_URL") != "" {
		storage = store.NewPostgresqlStore(store.NewPostgresqlClient(os.Getenv("POSTGRESQL_URL")))
		slog.Info("Storage initialised", "type", "postgresql")
	} else if os.Getenv("REDIS_URI") != "" {
		storage = store.NewRedisStore(store.NewRedisClient(os.Getenv("REDIS_URI"), os.Getenv("REDIS_PASSWORD")))
		slog.Info("Storage initialised", "type", "redis", "uri", os.Getenv("REDIS_URI"))
	} else {
		storage = store.NewDiskStore()
		slog.Info("Storage initialised", "type", "disk")
	}

	apiHandler := api.New(storage, staticContent)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/auth/device/code", apiHandler.StartAuth)
	mux.HandleFunc("GET /api/auth/device/poll", apiHandler.PollAuth)
	mux.HandleFunc("POST /api", apiHandler.WebhookHandler)
	mux.HandleFunc("POST /config", apiHandler.ConfigHandler)
	mux.HandleFunc("POST /logout", apiHandler.LogoutHandler)
	mux.Handle("GET /healthcheck", apiHandler.HealthcheckHandler())
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	mux.HandleFunc("GET /", apiHandler.RootHandler)

	var handler http.Handler = mux

	allowedHosts := cmp.Or(os.Getenv("ALLOWED_HOSTNAMES"), os.Getenv("REDIRECT_URI"))

	if allowedHosts != "" {
		handler = apiHandler.AllowedHostsHandler(allowedHosts)(handler)
	}

	handler = handlers.ProxyHeaders(handler)

	listen := cmp.Or(os.Getenv("LISTEN"), "0.0.0.0:8000")
	slog.Info("Server listening", "address", listen)
	if err := http.ListenAndServe(listen, handler); err != nil {
		slog.Error("Server crashed", "error", err)
		os.Exit(1)
	}
}

func setupLogging() {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Remove time attribute to avoid redundancy with host/container logs
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}

	if strings.ToLower(os.Getenv("LOG_LEVEL")) == "debug" {
		opts.Level = slog.LevelDebug
	}

	var handler slog.Handler
	if strings.ToLower(os.Getenv("JSON_LOGS")) == "true" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	slog.SetDefault(slog.New(handler))

	// Redirect standard log to slog as well
	log.SetOutput(&slogWriter{})
	log.SetFlags(0) // Remove standard log timestamps
}

type slogWriter struct{}

func (w *slogWriter) Write(p []byte) (n int, err error) {
	slog.Info(strings.TrimSpace(string(p)))
	return len(p), nil
}
