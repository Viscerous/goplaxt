package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

const (
	// CookieName is the name of the persistent user session cookie
	CookieName = "goplaxt_user"
)

// Pre-compiled regex for hostname validation
var hostnameCleanerRegex = regexp.MustCompile(`https://|http://|\s+`)

// SelfRoot returns the base URL for the application
func SelfRoot(r *http.Request) string {
	u, _ := url.Parse("")
	u.Host = r.Host
	u.Scheme = r.URL.Scheme
	u.Path = ""
	if u.Scheme == "" {
		u.Scheme = "http"
		proto := r.Header.Get("X-Forwarded-Proto")
		if proto == "https" {
			u.Scheme = "https"
		}
	}
	return u.String()
}

// isSecure checks if the request is over HTTPS
func (a *API) isSecure(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}

// setCookie sets a persistent user cookie
func (a *API) setCookie(w http.ResponseWriter, r *http.Request, userID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    userID,
		Path:     "/",
		Expires:  time.Now().Add(365 * 24 * time.Hour),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   a.isSecure(r),
	})
}

// clearCookie removes the user cookie
func (a *API) clearCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   a.isSecure(r),
	})
}

// getUserIDFromRequest gets user ID from query params or cookie
func getUserIDFromRequest(r *http.Request) string {
	// Check query params first (legacy)
	if id := r.URL.Query().Get("id"); id != "" {
		return id
	}
	// Check cookie
	if cookie, err := r.Cookie(CookieName); err == nil {
		return cookie.Value
	}
	return ""
}

// tokenResponse is the typed structure for Trakt OAuth responses
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	CreatedAt    int64  `json:"created_at"`
}

// extractTokenData safely extracts token data from Trakt auth response
func extractTokenData(result map[string]interface{}) (accessToken, refreshToken string, expiresIn, createdAt int64, ok bool) {
	// Re-marshal and unmarshal into typed struct for clean extraction
	data, err := json.Marshal(result)
	if err != nil {
		return "", "", 0, 0, false
	}

	var tr tokenResponse
	if err := json.Unmarshal(data, &tr); err != nil {
		return "", "", 0, 0, false
	}

	if tr.AccessToken == "" || tr.RefreshToken == "" {
		return "", "", 0, 0, false
	}

	return tr.AccessToken, tr.RefreshToken, tr.ExpiresIn, tr.CreatedAt, true
}
