package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/etherlabsio/healthcheck"
)

// AllowedHostsHandler creates middleware that restricts requests to specified hostnames
func (a *API) AllowedHostsHandler(allowedHostnames string) func(http.Handler) http.Handler {
	allowedHosts := strings.Split(hostnameCleanerRegex.ReplaceAllString(strings.ToLower(allowedHostnames), ""), ",")
	slog.Info("Configured allowed hostnames", "hosts", allowedHosts)

	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Always allow healthcheck
			if r.URL.EscapedPath() == "/healthcheck" {
				h.ServeHTTP(w, r)
				return
			}

			lcHost := strings.ToLower(r.Host)
			allowed := false
			for _, host := range allowedHosts {
				if lcHost == host {
					allowed = true
					break
				}
			}

			if !allowed {
				slog.Warn("Rejected request from unauthorised host", "host", lcHost, "allowed", allowedHosts)
				w.WriteHeader(http.StatusUnauthorized)
				w.Header().Set("Content-Type", "text/plain")
				fmt.Fprint(w, "Oh no!")
				return
			}

			h.ServeHTTP(w, r)
		})
	}
}

// HealthcheckHandler returns a health check handler
func (a *API) HealthcheckHandler() http.Handler {
	return healthcheck.Handler(
		healthcheck.WithTimeout(5*time.Second),
		healthcheck.WithChecker("storage", healthcheck.CheckerFunc(func(ctx context.Context) error {
			return a.Storage.Ping()
		})),
	)
}
