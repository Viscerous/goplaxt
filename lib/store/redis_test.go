package store

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
)

func TestRedisStore(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer s.Close()

	store := NewRedisStore(NewRedisClient(s.Addr(), ""))

	// Create user with config
	boolTrue := true
	config := Config{
		MovieScrobbleStart:   &boolTrue,
		MovieScrobbleStop:    &boolTrue,
		MovieRate:            &boolTrue,
		EpisodeScrobbleStart: &boolTrue,
		EpisodeScrobbleStop:  &boolTrue,
		EpisodeRate:          &boolTrue,
	}

	user := User{
		ID:             "test-id",
		Username:       "TestUser",
		PlexUsername:   "PlexTest",
		AccessToken:    "access123",
		RefreshToken:   "refresh123",
		TokenExpiresAt: time.Now().Add(1 * time.Hour).Truncate(time.Second),
		Config:         config,
		Store:          store,
	}

	// Write user
	err = store.WriteUser(user)
	assert.NoError(t, err)

	// Read user
	actual := store.GetUser("test-id")
	assert.NotNil(t, actual)
	assert.Equal(t, user.ID, actual.ID)
	assert.Equal(t, user.Username, actual.Username)
	assert.Equal(t, user.PlexUsername, actual.PlexUsername)
	assert.True(t, actual.Config.GetMovieScrobbleStart())
	assert.True(t, actual.Config.GetEpisodeRate())
	assert.True(t, actual.IsConfigured())

	// Test GetUserByUsername
	foundByName := store.GetUserByUsername("TestUser")
	assert.NotNil(t, foundByName)
	assert.Equal(t, user.ID, foundByName.ID)

	// Test case insensitivity
	foundLower := store.GetUserByUsername("testuser")
	assert.NotNil(t, foundLower)
	assert.Equal(t, user.ID, foundLower.ID)

	// Test DeleteUser
	deleted := store.DeleteUser("test-id")
	assert.True(t, deleted)

	notFound := store.GetUser("test-id")
	assert.Nil(t, notFound)
}

func TestRedisPing(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer s.Close()

	store := NewRedisStore(NewRedisClient(s.Addr(), ""))
	assert.NoError(t, store.Ping())
}
