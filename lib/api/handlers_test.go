package api

import (
	"testing/fstest"

	"github.com/gorilla/handlers"
	"github.com/stretchr/testify/assert"

	"errors"

	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/viscerous/goplaxt/lib/store"
)

func TestSelfRoot(t *testing.T) {
	var (
		r   *http.Request
		err error
	)

	// Test Default
	r, err = http.NewRequest("GET", "/authorize", nil)
	r.Host = "foo.bar"
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "http://foo.bar", SelfRoot(r))

	// Test Manual forwarded proto
	r, err = http.NewRequest("GET", "/validate", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Host = "foo.bar"
	r.Header.Set("X-Forwarded-Proto", "https")
	assert.Equal(t, "https://foo.bar", SelfRoot(r))

	// Test ProxyHeader handler
	rr := httptest.NewRecorder()
	r, err = http.NewRequest("GET", "/validate", nil)
	r.Header.Set("X-Forwarded-Host", "foo.bar")
	r.Header.Set("X-Forwarded-Proto", "https")
	handlers.ProxyHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(rr, r)
	assert.Equal(t, "https://foo.bar", SelfRoot(r))
}

func TestAllowedHostsHandler_single_hostname(t *testing.T) {
	// API struct doesn't need storage for this test, but we'll provide nil
	api := &API{}
	f := api.AllowedHostsHandler("foo.bar")

	rr := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Host = "foo.bar"

	f(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(rr, r)
	assert.Equal(t, http.StatusOK, rr.Result().StatusCode)
}

func TestAllowedHostsHandler_multiple_hostnames(t *testing.T) {
	api := &API{}
	f := api.AllowedHostsHandler("foo.bar, bar.foo")

	rr := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Host = "bar.foo"

	f(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(rr, r)
	assert.Equal(t, http.StatusOK, rr.Result().StatusCode)
}

func TestAllowedHostsHandler_mismatch_hostname(t *testing.T) {
	api := &API{}
	f := api.AllowedHostsHandler("unknown.host")

	rr := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Host = "known.host"

	f(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(rr, r)
	assert.Equal(t, http.StatusUnauthorized, rr.Result().StatusCode)
}

func TestAllowedHostsHandler_alwaysAllowHealthcheck(t *testing.T) {
	api := New(&MockSuccessStore{}, fstest.MapFS{"static/index.html": {Data: []byte("TEST")}})
	f := api.AllowedHostsHandler("unknown.host")

	rr := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/healthcheck", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Host = "known.host"

	f(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(rr, r)
	assert.Equal(t, http.StatusOK, rr.Result().StatusCode)
}

type MockSuccessStore struct{}

func (s MockSuccessStore) Ping() error                     { return nil }
func (s MockSuccessStore) WriteUser(user store.User) error { return nil }
func (s MockSuccessStore) GetUser(id string) *store.User {
	if id == "user123" {
		return &store.User{ID: "user123", Username: "traktuser", Store: s}
	}
	return nil
}
func (s MockSuccessStore) GetUserByUsername(username string) *store.User { return nil }
func (s MockSuccessStore) DeleteUser(id string) bool                     { return true }

func TestAPI_Multipart(t *testing.T) {
	api := New(&MockSuccessStore{}, fstest.MapFS{"static/index.html": {Data: []byte("TEST")}})

	// Create a multipart form request
	body := "{\"event\": \"media.play\", \"Account\": {\"title\": \"traktuser\"}}"

	// manually construct multipart to avoid complex writer setup for a simple test
	boundary := "---test-boundary"
	payload := "--" + boundary + "\r\n" +
		"Content-Disposition: form-data; name=\"payload\"\r\n" +
		"\r\n" +
		body + "\r\n" +
		"--" + boundary + "--\r\n"

	r, err := http.NewRequest("POST", "/api?id=user123", strings.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Content-Type", "multipart/form-data; boundary="+boundary)

	rr := httptest.NewRecorder()
	api.WebhookHandler(rr, r)

	assert.Equal(t, http.StatusOK, rr.Result().StatusCode)
	assert.Contains(t, rr.Body.String(), "processing in background")
}

type MockFailStore struct{}

func (s MockFailStore) Ping() error                                   { return errors.New("OH NO") }
func (s MockFailStore) WriteUser(user store.User) error               { return errors.New("OH NO") }
func (s MockFailStore) GetUser(id string) *store.User                 { panic(errors.New("OH NO")) }
func (s MockFailStore) GetUserByUsername(username string) *store.User { panic(errors.New("OH NO")) }
func (s MockFailStore) DeleteUser(id string) bool                     { return false }

func TestHealthcheck(t *testing.T) {
	var rr *httptest.ResponseRecorder

	r, err := http.NewRequest("GET", "/healthcheck", nil)
	if err != nil {
		t.Fatal(err)
	}

	api := New(&MockSuccessStore{}, fstest.MapFS{"static/index.html": {Data: []byte("TEST")}})
	rr = httptest.NewRecorder()
	http.Handler(api.HealthcheckHandler()).ServeHTTP(rr, r)
	assert.Equal(t, http.StatusOK, rr.Result().StatusCode)
	assert.Equal(t, "{\"status\":\"OK\"}\n", rr.Body.String())

	apiFail := New(&MockFailStore{}, fstest.MapFS{"static/index.html": {Data: []byte("TEST")}})
	rr = httptest.NewRecorder()
	http.Handler(apiFail.HealthcheckHandler()).ServeHTTP(rr, r)
	assert.Equal(t, http.StatusServiceUnavailable, rr.Result().StatusCode)
	assert.Equal(t, "{\"status\":\"Service Unavailable\",\"errors\":{\"storage\":\"OH NO\"}}\n", rr.Body.String())
}

type SpyStore struct {
	MockSuccessStore
	DeletedUsers []string
}

func (s *SpyStore) DeleteUser(id string) bool {
	s.DeletedUsers = append(s.DeletedUsers, id)
	return true
}

func TestLogoutHandler(t *testing.T) {
	spyStore := &SpyStore{}
	api := New(spyStore, fstest.MapFS{"static/index.html": {Data: []byte("TEST")}})

	// 1. Valid Logout
	r, _ := http.NewRequest("POST", "/logout", nil)
	r.AddCookie(&http.Cookie{Name: "goplaxt_user", Value: "user123"})
	rr := httptest.NewRecorder()

	api.LogoutHandler(rr, r)

	assert.Equal(t, http.StatusSeeOther, rr.Result().StatusCode)
	assert.Equal(t, "/", rr.Result().Header.Get("Location"))

	// Verify Cookie Cleared
	cookies := rr.Result().Cookies()
	assert.NotEmpty(t, cookies)
	found := false
	for _, c := range cookies {
		if c.Name == "goplaxt_user" {
			found = true
			assert.Equal(t, "", c.Value)
			assert.True(t, c.Expires.Before(time.Now())) // Should be expired
			break
		}
	}
	assert.True(t, found, "Logout cookie should be set")

	// Verify Store Deletion
	assert.Contains(t, spyStore.DeletedUsers, "user123")
	assert.Equal(t, 1, len(spyStore.DeletedUsers))

	// 2. Invalid Method
	r, _ = http.NewRequest("GET", "/logout", nil)
	rr = httptest.NewRecorder()
	api.LogoutHandler(rr, r)
	assert.Equal(t, http.StatusMethodNotAllowed, rr.Result().StatusCode)

	// 3. No Cookie
	r, _ = http.NewRequest("POST", "/logout", nil)
	rr = httptest.NewRecorder()
	api.LogoutHandler(rr, r)
	assert.Equal(t, http.StatusSeeOther, rr.Result().StatusCode) // Redirects home
	// Should not delete anything
	assert.Equal(t, 1, len(spyStore.DeletedUsers)) // Count same as before
}
