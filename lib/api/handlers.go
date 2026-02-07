package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/viscerous/goplaxt/lib/store"
	"github.com/viscerous/goplaxt/lib/trakt"
	"github.com/xanderstrike/plexhooks"
)

// API is the main application handler
type API struct {
	Storage           store.Store
	UserLocks         sync.Map
	AuthoriseTemplate *template.Template
}

// New creates a new API instance
func New(storage store.Store, content fs.FS) *API {
	tpl, err := template.ParseFS(content, "static/index.html")
	if err != nil {
		panic(fmt.Errorf("failed to parse templates: %w", err))
	}

	return &API{
		Storage:           storage,
		AuthoriseTemplate: tpl,
	}
}

// AuthorisePage holds data for the main page template
type AuthorisePage struct {
	SelfRoot    string
	Authorised  bool
	URL         string
	User        store.User
	CurrentStep int // 1=Auth, 2=Webhook, 3=Config, 4=Dashboard
}

// RootHandler renders the main page
func (a *API) RootHandler(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromRequest(r)

	// Set cookie if ID was in query params
	if id := r.URL.Query().Get("id"); id != "" && id == userID {
		a.setCookie(w, r, userID)
	}

	var user *store.User
	var apiURL string
	authorised := false

	if userID != "" {
		user = a.Storage.GetUser(userID)
		if user != nil {
			authorised = true
			apiURL = fmt.Sprintf("%s/api?id=%s", SelfRoot(r), user.ID)
		}
	}

	currentStep := determineStep(authorised, user)

	data := AuthorisePage{
		SelfRoot:    SelfRoot(r),
		Authorised:  authorised,
		URL:         apiURL,
		CurrentStep: currentStep,
	}

	if authorised && user != nil {
		data.User = *user
	}

	if err := a.AuthoriseTemplate.Execute(w, data); err != nil {
		slog.Error("Failed to render template", "error", err)
	}
}

// determineStep calculates the current wizard step
func determineStep(authorised bool, user *store.User) int {
	if !authorised || user == nil || user.AccessToken == "" {
		return 1
	}

	// Dashboard requires both PlexUsername and config
	if user.PlexUsername != "" && user.IsConfigured() {
		return 4
	}
	if user.PlexUsername != "" {
		return 3 // Has username but missing config
	}
	return 2 // Needs webhook setup
}

// WebhookHandler handles incoming Plex webhook events
func (a *API) WebhookHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("id")
	if userID == "" {
		http.Error(w, "Missing id", http.StatusBadRequest)
		return
	}
	slog.Info("Webhook received", "user_id", userID)

	// Validate user exists
	initialUser := a.Storage.GetUser(userID)
	if initialUser == nil {
		slog.Warn("Webhook rejected: user not found", "id", userID)
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode("user not found")
		return
	}

	// Extract payload
	payload, err := a.extractPayload(r)
	if err != nil {
		slog.Debug("No payload in request", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Parse webhook
	plexEvent, err := plexhooks.ParseWebhook(payload)
	if err != nil {
		slog.Error("Error parsing webhook", "error", err)
		http.Error(w, "Invalid Webhook", http.StatusBadRequest)
		return
	}

	// Respond immediately
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode("processing in background")

	// Process asynchronously
	go a.processWebhook(userID, payload, plexEvent)
}

// extractPayload extracts the webhook payload from the request
func (a *API) extractPayload(r *http.Request) ([]byte, error) {
	contentType := r.Header.Get("Content-Type")

	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			return nil, fmt.Errorf("invalid multipart form: %w", err)
		}
		payload := []byte(r.FormValue("payload"))
		if len(payload) == 0 {
			return nil, fmt.Errorf("missing payload")
		}
		return payload, nil
	}

	// Raw JSON fallback
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}
	if len(payload) == 0 {
		return nil, fmt.Errorf("missing payload")
	}
	return payload, nil
}

// processWebhook handles webhook processing in the background
func (a *API) processWebhook(userID string, payload []byte, plexEvent plexhooks.PlexResponse) {
	ctx := context.Background()

	// Per-user mutex
	mutex, _ := a.UserLocks.LoadOrStore(userID, &sync.Mutex{})
	mtx := mutex.(*sync.Mutex)
	mtx.Lock()
	defer mtx.Unlock()

	// Reload user for latest state
	user := a.Storage.GetUser(userID)
	if user == nil {
		slog.Warn("User disappeared during processing", "user_id", userID)
		return
	}

	// Refresh token if expired
	if time.Now().After(user.TokenExpiresAt) {
		if err := a.refreshToken(user); err != nil {
			slog.Error("Token refresh failed", "user_id", user.ID, "error", err)
			return
		}
	}

	// Match user
	if user.PlexUsername == "" || !strings.EqualFold(plexEvent.Account.Title, user.PlexUsername) {
		expected := user.PlexUsername
		if expected == "" {
			expected = user.Username
		}
		slog.Debug("Plex user mismatch", "got", plexEvent.Account.Title, "expected", expected)
		return
	}

	client := &trakt.RealTraktClient{}
	trakt.Handle(ctx, client, plexEvent, payload, *user)
}

// refreshToken refreshes an expired Trakt token
func (a *API) refreshToken(user *store.User) error {
	slog.Info("Refreshing Trakt token", "user_id", user.ID)

	result, err := trakt.AuthRequest("", "", user.RefreshToken, "refresh_token")
	if err != nil {
		if errors.Is(err, trakt.ErrInvalidToken) {
			slog.Warn("Trakt session revoked, clearing tokens", "user_id", user.ID)
			user.AccessToken = ""
			user.RefreshToken = ""
			user.Save()
		}
		return err
	}

	accessToken, refreshToken, expiresIn, createdAt, ok := extractTokenData(result)
	if !ok {
		return fmt.Errorf("invalid refresh response")
	}

	user.UpdateUser(accessToken, refreshToken, expiresIn, createdAt)
	return nil
}
