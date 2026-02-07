package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/viscerous/goplaxt/lib/store"
	"github.com/viscerous/goplaxt/lib/trakt"
)

// StartAuth initiates device code authentication flow
func (a *API) StartAuth(w http.ResponseWriter, r *http.Request) {
	codeData, err := trakt.GetDeviceCode()
	if err != nil {
		slog.Error("Failed to get device code", "error", err)
		http.Error(w, "Failed to start auth", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(codeData); err != nil {
		slog.Error("Failed to encode device code response", "error", err)
	}
}

// PollAuth polls for device authentication completion
func (a *API) PollAuth(w http.ResponseWriter, r *http.Request) {
	deviceCode := r.URL.Query().Get("device_code")
	if deviceCode == "" {
		http.Error(w, "Missing device_code", http.StatusBadRequest)
		return
	}

	result, err := trakt.PollDeviceToken(deviceCode)
	if err != nil {
		slog.Warn("Poll failed", "device_code", deviceCode, "error", err)
		switch err.Error() {
		case "invalid code":
			http.Error(w, "Invalid code", http.StatusNotFound)
		case "expired code":
			http.Error(w, "Expired code", http.StatusGone)
		case "already used code":
			http.Error(w, "Code already used", http.StatusConflict)
		default:
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	if result == nil {
		// Pending
		w.WriteHeader(http.StatusAccepted)
		return
	}

	// Success - extract token data
	accessToken, refreshToken, expiresIn, createdAt, ok := extractTokenData(result)
	if !ok {
		slog.Error("Invalid token response from Trakt", "result_keys", result)
		http.Error(w, "Invalid token response", http.StatusInternalServerError)
		return
	}

	// Fetch user profile
	profile, err := trakt.GetUserProfile(accessToken)
	if err != nil {
		slog.Error("Failed to fetch user profile", "error", err)
		http.Error(w, "Failed to fetch user profile", http.StatusInternalServerError)
		return
	}

	username := extractUsername(profile)
	if username == "" {
		slog.Error("Username not found in profile", "profile_keys", profile)
		http.Error(w, "Could not determine username", http.StatusInternalServerError)
		return
	}

	// Find or create user
	var user store.User
	existingUser := a.Storage.GetUserByUsername(username)
	if existingUser != nil {
		slog.Info("Updating existing user", "id", existingUser.ID, "username", username)
		existingUser.UpdateUser(accessToken, refreshToken, expiresIn, createdAt)
		user = *existingUser
	} else {
		// Attempt recovery from cookie
		if recovered := a.tryRecoverUser(r, username, accessToken, refreshToken, expiresIn, createdAt); recovered != nil {
			user = *recovered
		} else {
			user = store.NewUser(username, accessToken, refreshToken, expiresIn, createdAt, a.Storage)
		}
	}

	a.setCookie(w, r, user.ID)
	w.WriteHeader(http.StatusOK)
}

// extractUsername safely extracts username from Trakt profile
func extractUsername(profile map[string]interface{}) string {
	if u, ok := profile["username"].(string); ok {
		return u
	}
	if uObj, ok := profile["user"].(map[string]interface{}); ok {
		if u, ok := uObj["username"].(string); ok {
			return u
		}
	}
	return ""
}

// tryRecoverUser attempts to recover a user from cookie
func (a *API) tryRecoverUser(r *http.Request, username, accessToken, refreshToken string, expiresIn, createdAt int64) *store.User {
	cookie, err := r.Cookie("goplaxt_user")
	if err != nil || cookie.Value == "" {
		return nil
	}

	if a.Storage.GetUser(cookie.Value) != nil {
		return nil // User exists, no recovery needed
	}

	slog.Info("Recovering lost user from cookie", "id", cookie.Value, "username", username)
	user := store.NewUserWithID(cookie.Value, username, accessToken, refreshToken, expiresIn, createdAt, a.Storage)
	return &user
}
