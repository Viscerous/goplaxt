package trakt

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/viscerous/goplaxt/lib/config"
)

func TestRealTraktClient_DoRequest_Headers(t *testing.T) {
	// Setup config
	config.TraktClientId = "test-client-id"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "test-client-id", r.Header.Get("trakt-api-key"))
		assert.Equal(t, "2", r.Header.Get("trakt-api-version"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	// Override BaseURL for testing
	originalBaseURL := BaseURL
	BaseURL = server.URL
	defer func() { BaseURL = originalBaseURL }()

	client := &RealTraktClient{}

	// Test ScrobbleRequest (uses doRequest internally)
	_, err := client.ScrobbleRequest(context.Background(), "start", []byte("{}"), "test-token")
	assert.NoError(t, err)
}

func TestRealTraktClient_DoRequest_Retry(t *testing.T) {
	// Setup config
	config.TraktClientId = "test-client-id"

	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	// Override BaseURL for testing
	originalBaseURL := BaseURL
	BaseURL = server.URL
	defer func() { BaseURL = originalBaseURL }()

	client := &RealTraktClient{}

	_, err := client.ScrobbleRequest(context.Background(), "start", []byte("{}"), "test-token")
	assert.NoError(t, err)
	assert.Equal(t, 3, attempts, "Should have retried 3 times")
}
