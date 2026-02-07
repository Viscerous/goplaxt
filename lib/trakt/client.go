package trakt

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/viscerous/goplaxt/lib/config"
	"golang.org/x/time/rate"
)

var (
	BaseURL         = "https://api.trakt.tv"
	ErrInvalidToken = errors.New("invalid_token")
)

// Package-level HTTP client for connection pooling and reuse
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

// Rate limiter: Trakt allows 1000 calls per 5 minutes = ~3.3/sec
// We use 2/sec with burst of 5 to be conservative
var rateLimiter = rate.NewLimiter(rate.Limit(2), 5)

type Client interface {
	MakeRequest(ctx context.Context, url string) ([]byte, error)
	ScrobbleRequest(ctx context.Context, action string, body []byte, token string) ([]byte, error)
	SyncRequest(ctx context.Context, endpoint string, body []byte, token string) ([]byte, error)
	DeleteCheckin(ctx context.Context, token string) error
}

type RealTraktClient struct{}

func (c *RealTraktClient) MakeRequest(ctx context.Context, url string) ([]byte, error) {
	return makeRequest(ctx, url)
}

func (c *RealTraktClient) ScrobbleRequest(ctx context.Context, action string, body []byte, token string) ([]byte, error) {
	return scrobbleRequest(ctx, action, body, token)
}

func (c *RealTraktClient) SyncRequest(ctx context.Context, endpoint string, body []byte, token string) ([]byte, error) {
	return syncRequest(ctx, endpoint, body, token)
}

func (c *RealTraktClient) DeleteCheckin(ctx context.Context, token string) error {
	return deleteCheckin(ctx, token)
}

// AuthRequest authorises the connection with Trakt
func AuthRequest(root, code, refreshToken, grantType string) (map[string]interface{}, error) {
	values := map[string]string{
		"code":          code,
		"refresh_token": refreshToken,
		"client_id":     config.TraktClientId,
		"client_secret": config.TraktClientSecret,
		"redirect_uri":  fmt.Sprintf("%s/authorize", root),
		"grant_type":    grantType,
	}

	result, err := doPost(context.Background(), "/oauth/token", values)
	if err != nil {
		if strings.Contains(err.Error(), "400") || strings.Contains(err.Error(), "401") {
			return nil, ErrInvalidToken
		}
		return nil, err
	}
	return result, nil
}

// GetDeviceCode initiates the Device Flow
func GetDeviceCode() (map[string]interface{}, error) {
	values := map[string]string{
		"client_id": config.TraktClientId,
	}
	return doPost(context.Background(), "/oauth/device/code", values)
}

// PollDeviceToken polls for the user token
func PollDeviceToken(deviceCode string) (map[string]interface{}, error) {
	values := map[string]string{
		"code":          deviceCode,
		"client_id":     config.TraktClientId,
		"client_secret": config.TraktClientSecret,
	}
	jsonValue, err := json.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	var lastErr error
	for i := 0; i < 3; i++ {
		resp, err := http.Post(fmt.Sprintf("%s/oauth/device/token", BaseURL), "application/json", bytes.NewBuffer(jsonValue))
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(i+1) * time.Second)
			continue
		}
		defer resp.Body.Close()

		// Special handling for polling status codes
		if resp.StatusCode == http.StatusBadRequest {
			return nil, nil // Pending (not yet authorised)
		}
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("invalid code")
		}
		if resp.StatusCode == http.StatusConflict {
			return nil, fmt.Errorf("already used code")
		}
		if resp.StatusCode == http.StatusGone {
			return nil, fmt.Errorf("expired code")
		}

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			time.Sleep(time.Duration(i+1) * time.Second)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		return result, nil
	}

	return nil, fmt.Errorf("request failed after 3 attempts: %w", lastErr)
}

// doPost is a internal helper for OAuth requests with retries
func doPost(ctx context.Context, endpoint string, values map[string]string) (map[string]interface{}, error) {
	jsonValue, err := json.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	apiUrl := fmt.Sprintf("%s/%s", BaseURL, strings.TrimPrefix(endpoint, "/"))

	var lastErr error
	for i := 0; i < 3; i++ {
		req, err := http.NewRequestWithContext(ctx, "POST", apiUrl, bytes.NewBuffer(jsonValue))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(i+1) * time.Second)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			time.Sleep(time.Duration(i+1) * time.Second)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("trakt api returned bad status: %d", resp.StatusCode)
		}

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return result, nil
	}
	return nil, fmt.Errorf("request failed after 3 attempts: %w", lastErr)
}

// GetUserProfile fetches the authenticated user's profile
func GetUserProfile(token string) (map[string]interface{}, error) {
	respBody, err := doRequest(context.Background(), "GET", fmt.Sprintf("%s/users/me", BaseURL), nil, token)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

func makeRequest(ctx context.Context, url string) ([]byte, error) {
	// If url starts with http, use it, otherwise prepend BaseURL
	if !strings.HasPrefix(url, "http") {
		url = fmt.Sprintf("%s/%s", BaseURL, strings.TrimPrefix(url, "/"))
	}
	return doRequest(ctx, "GET", url, nil, "")
}

func scrobbleRequest(ctx context.Context, action string, body []byte, accessToken string) ([]byte, error) {
	apiUrl := fmt.Sprintf("%s/scrobble/%s", BaseURL, action)
	return doRequest(ctx, "POST", apiUrl, body, accessToken)
}

func syncRequest(ctx context.Context, endpoint string, body []byte, accessToken string) ([]byte, error) {
	apiUrl := fmt.Sprintf("%s/sync/%s", BaseURL, endpoint)
	return doRequest(ctx, "POST", apiUrl, body, accessToken)
}

func deleteCheckin(ctx context.Context, accessToken string) error {
	_, err := doRequest(ctx, "DELETE", fmt.Sprintf("%s/checkin", BaseURL), nil, accessToken)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return nil
		}
		return err
	}
	return nil
}

func doRequest(ctx context.Context, method, url string, body []byte, accessToken string) ([]byte, error) {
	var lastErr error
	for i := 0; i < 3; i++ {
		// Rate limiting
		if err := rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limiter cancelled: %w", err)
		}

		var req *http.Request
		var err error

		if len(body) > 0 {
			req, err = http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(body))
		} else {
			req, err = http.NewRequestWithContext(ctx, method, url, nil)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Add("Content-Type", "application/json")
		if accessToken != "" {
			req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
		}
		req.Header.Add("trakt-api-version", "2")
		req.Header.Add("trakt-api-key", config.TraktClientId)

		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
			slog.Warn("Trakt request failed", "attempt", i+1, "method", method, "url", url, "error", err)
			time.Sleep(time.Duration(i+1) * time.Second)
			continue
		}

		defer resp.Body.Close()

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			slog.Warn("Trakt server error", "status", resp.StatusCode, "attempt", i+1)
			time.Sleep(time.Duration(i+1) * time.Second)
			continue
		}

		if resp.StatusCode >= 400 {
			respBody, _ := io.ReadAll(resp.Body)
			slog.Error("Trakt client error", "status", resp.StatusCode, "url", url, "response", string(respBody))
			return nil, fmt.Errorf("trakt api returned bad status: %d", resp.StatusCode)
		}

		if resp.StatusCode == http.StatusNoContent {
			return nil, nil
		}

		return io.ReadAll(resp.Body)
	}

	return nil, fmt.Errorf("request failed after 3 attempts: %w", lastErr)
}
