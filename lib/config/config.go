package config

import (
	"cmp"
	"log/slog"
	"os"
	"strings"
)

var TraktClientId string = getConfig("TRAKT_ID")
var TraktClientSecret string = getConfig("TRAKT_SECRET")

func getConfig(name string) string {
	return cmp.Or(os.Getenv(name), readSecretFile(name+"_FILE"))
}

func readSecretFile(name string) string {
	path := os.Getenv(name)
	if path == "" {
		return ""
	}
	file, err := os.ReadFile(path)
	if err != nil {
		slog.Warn("Failed to read secret file", "env_var", name, "path", path, "error", err)
		return ""
	}
	return strings.TrimSpace(string(file))
}
