package api

import (
	"net/http"
	"net/url"
	"regexp"
	"time"
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
		Name:     "goplaxt_user",
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
		Name:     "goplaxt_user",
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
	if cookie, err := r.Cookie("goplaxt_user"); err == nil {
		return cookie.Value
	}
	return ""
}

// extractTokenData safely extracts token data from Trakt auth response
func extractTokenData(result map[string]interface{}) (accessToken, refreshToken string, expiresIn, createdAt int64, ok bool) {
	var t interface{}
	t, ok = result["access_token"]
	if !ok {
		return
	}
	accessToken, ok = t.(string)
	if !ok {
		return
	}
	t, ok = result["refresh_token"]
	if !ok {
		return
	}
	refreshToken, ok = t.(string)
	if !ok {
		return
	}
	t, ok = result["expires_in"]
	if !ok {
		return
	}
	if exp, isFloat := t.(float64); isFloat {
		expiresIn = int64(exp)
	} else {
		ok = false
		return
	}
	t, ok = result["created_at"]
	if !ok {
		return
	}
	if created, isFloat := t.(float64); isFloat {
		createdAt = int64(created)
	} else {
		ok = false
		return
	}
	return
}
