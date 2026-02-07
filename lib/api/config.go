package api

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/viscerous/goplaxt/lib/store"
)

// ConfigHandler handles user configuration updates
func (a *API) ConfigHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		slog.Error("Error parsing form", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	userID := r.Form.Get("id")
	if userID == "" {
		userID = getUserIDFromRequest(r)
	}

	user := a.Storage.GetUser(userID)
	if user == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Smart default: use Trakt username if Plex username empty
	plexUsername := strings.TrimSpace(r.Form.Get("plex_username"))
	if plexUsername == "" {
		plexUsername = user.Username
	}

	// Helper for pointers
	boolPtr := func(b bool) *bool { return &b }

	config := store.Config{
		MovieScrobbleStart:   boolPtr(r.Form.Get("movie_scrobble_start") == "on"),
		MovieScrobbleStop:    boolPtr(r.Form.Get("movie_scrobble_stop") == "on"),
		MovieRate:            boolPtr(r.Form.Get("movie_rate") == "on"),
		MovieCollection:      boolPtr(r.Form.Get("movie_collection") == "on"),
		EpisodeScrobbleStart: boolPtr(r.Form.Get("episode_scrobble_start") == "on"),
		EpisodeScrobbleStop:  boolPtr(r.Form.Get("episode_scrobble_stop") == "on"),
		EpisodeRate:          boolPtr(r.Form.Get("episode_rate") == "on"),
		EpisodeCollection:    boolPtr(r.Form.Get("episode_collection") == "on"),
		ShowRate:             boolPtr(r.Form.Get("show_rate") == "on"),
		SeasonRate:           boolPtr(r.Form.Get("season_rate") == "on"),
	}

	user.UpdateConfiguration(config, plexUsername)
	slog.Info("User configuration updated", "user_id", userID)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// LogoutHandler handles user logout and data deletion
func (a *API) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	cookie, err := r.Cookie("goplaxt_user")
	if err != nil || cookie.Value == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	slog.Info("Logging out and deleting user", "id", cookie.Value)
	a.Storage.DeleteUser(cookie.Value)
	a.clearCookie(w, r)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
