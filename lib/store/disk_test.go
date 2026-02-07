package store

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiskStore(t *testing.T) {
	// Setup
	os.RemoveAll("keystore")
	defer os.RemoveAll("keystore")

	store := NewDiskStore()

	// Create user with config
	user := NewUser("TestUser", "Access123", "Refresh123", 3600, 1000, store)
	boolTrue := true
	user.Config = Config{
		MovieScrobbleStart:   &boolTrue,
		MovieScrobbleStop:    &boolTrue,
		MovieRate:            &boolTrue,
		EpisodeScrobbleStart: &boolTrue,
		EpisodeScrobbleStop:  &boolTrue,
		EpisodeRate:          &boolTrue,
	}
	user.PlexUsername = "PlexTest"
	err := store.WriteUser(user)
	assert.NoError(t, err)

	// Test GetUser
	found := store.GetUser(user.ID)
	assert.NotNil(t, found)
	assert.Equal(t, user.Username, found.Username)
	assert.Equal(t, user.PlexUsername, found.PlexUsername)
	assert.True(t, found.Config.GetMovieScrobbleStart())
	assert.True(t, found.IsConfigured())

	// Test GetUserByUsername
	foundByName := store.GetUserByUsername("TestUser")
	assert.NotNil(t, foundByName)
	assert.Equal(t, user.ID, foundByName.ID)

	// Test case insensitivity
	foundLower := store.GetUserByUsername("testuser")
	assert.NotNil(t, foundLower)
	assert.Equal(t, user.ID, foundLower.ID)

	// Test DeleteUser
	deleted := store.DeleteUser(user.ID)
	assert.True(t, deleted)

	// Verify deleted
	notFound := store.GetUser(user.ID)
	assert.Nil(t, notFound)
}

func TestDiskStoreIndex(t *testing.T) {
	os.RemoveAll("keystore")
	defer os.RemoveAll("keystore")

	store := NewDiskStore()
	user := NewUser("IndexUser", "Access", "Refresh", 3600, 1000, store)

	// Index should exist after write
	found := store.GetUserByUsername("IndexUser")
	assert.NotNil(t, found)
	assert.Equal(t, user.ID, found.ID)
}
