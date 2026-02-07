package store

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestPostgresqlStore(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error opening stub database: %s", err)
	}
	defer db.Close()

	store := NewPostgresqlStore(db)
	fixedTime := time.Date(2025, 3, 28, 22, 30, 55, 0, time.UTC)
	configJSON := []byte(`{"movie_scrobble_start":true,"movie_scrobble_stop":true,"movie_rate":true,"episode_scrobble_start":true,"episode_scrobble_stop":true,"episode_rate":true}`)

	// Test WriteUser
	mock.ExpectExec("INSERT INTO users").WillReturnResult(sqlmock.NewResult(1, 1))

	boolTrue := true
	user := User{
		ID:             "test-id",
		Username:       "TestUser",
		PlexUsername:   "PlexTest",
		AccessToken:    "access123",
		RefreshToken:   "refresh123",
		TokenExpiresAt: fixedTime,
		Config: Config{
			MovieScrobbleStart:   &boolTrue,
			MovieScrobbleStop:    &boolTrue,
			MovieRate:            &boolTrue,
			EpisodeScrobbleStart: &boolTrue,
			EpisodeScrobbleStop:  &boolTrue,
			EpisodeRate:          &boolTrue,
		},
		Store: store,
	}

	err = store.WriteUser(user)
	assert.NoError(t, err)

	// Test GetUser
	mock.ExpectQuery("SELECT .+ FROM users WHERE id = ").WithArgs("test-id").WillReturnRows(
		sqlmock.NewRows([]string{"id", "username", "plex_username", "access_token", "refresh_token", "token_expires_at", "config"}).
			AddRow("test-id", "TestUser", "PlexTest", "access123", "refresh123", fixedTime, configJSON),
	)

	actual := store.GetUser("test-id")
	assert.NotNil(t, actual)
	assert.Equal(t, "TestUser", actual.Username)
	assert.True(t, actual.Config.GetMovieScrobbleStart())
	assert.True(t, actual.IsConfigured())

	// Verify all expectations met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresqlGetByUsername(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	defer db.Close()

	store := NewPostgresqlStore(db)
	fixedTime := time.Now()
	configJSON := []byte(`{}`)

	mock.ExpectQuery("SELECT id FROM users WHERE").WithArgs("testuser").WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow("test-id"),
	)
	mock.ExpectQuery("SELECT .+ FROM users WHERE id = ").WithArgs("test-id").WillReturnRows(
		sqlmock.NewRows([]string{"id", "username", "plex_username", "access_token", "refresh_token", "token_expires_at", "config"}).
			AddRow("test-id", "TestUser", "", "access", "refresh", fixedTime, configJSON),
	)

	actual := store.GetUserByUsername("testuser")
	assert.NotNil(t, actual)
	assert.Equal(t, "test-id", actual.ID)

	assert.NoError(t, mock.ExpectationsWereMet())
}
