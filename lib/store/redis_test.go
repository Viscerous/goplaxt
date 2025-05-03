package store

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
)

func TestLoadingUser(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer s.Close()

	store := NewRedisStore(NewRedisClient(s.Addr(), ""))

	s.HSet("goplaxt:user:id123", "username", "halkeye")
	s.HSet("goplaxt:user:id123", "access", "access123")
	s.HSet("goplaxt:user:id123", "refresh", "refresh123")
	s.HSet("goplaxt:user:id123", "expires_at", "2025-03-28T22:30:55Z")

	expected, err := json.Marshal(&User{
		ID:             "id123",
		Username:       "halkeye",
		AccessToken:    "access123",
		RefreshToken:   "refresh123",
		TokenExpiresAt: time.Date(2025, 03, 28, 22, 30, 55, 0, time.UTC),
		Store:          store,
	})
	actual, err := json.Marshal(store.GetUser("id123"))

	assert.EqualValues(t, string(expected), string(actual))
}

func TestSavingUser(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer s.Close()

	store := NewRedisStore(NewRedisClient(s.Addr(), ""))
	originalUser := &User{
		ID:             "id123",
		Username:       "halkeye",
		AccessToken:    "access123",
		RefreshToken:   "refresh123",
		TokenExpiresAt: time.Date(2025, 03, 28, 22, 30, 55, 0, time.UTC),
		Store:          store,
	}

	originalUser.save()

	assert.Equal(t, s.HGet("goplaxt:user:id123", "username"), "halkeye")
	assert.Equal(t, s.HGet("goplaxt:user:id123", "access"), "access123")
	assert.Equal(t, s.HGet("goplaxt:user:id123", "refresh"), "refresh123")
	assert.Equal(t, s.HGet("goplaxt:user:id123", "expires_at"), "2025-03-28T22:30:55Z")

	expected, err := json.Marshal(originalUser)
	actual, err := json.Marshal(store.GetUser("id123"))

	assert.EqualValues(t, string(expected), string(actual))
}

func TestPing(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer s.Close()

	store := NewRedisStore(NewRedisClient(s.Addr(), ""))
	assert.Equal(t, store.Ping(context.TODO()), nil)
}
