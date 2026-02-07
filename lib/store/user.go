package store

import (
	"crypto/rand"
	"fmt"
	"log/slog"
	"time"
)

// Store is the interface for all storage backends
type Store interface {
	WriteUser(user User) error
	GetUser(id string) *User
	GetUserByUsername(username string) *User
	DeleteUser(id string) bool
	Ping() error
}

// Config holds user preferences for Trakt synchronisation
type Config struct {
	MovieScrobbleStart   *bool `json:"movie_scrobble_start"`
	MovieScrobbleStop    *bool `json:"movie_scrobble_stop"`
	MovieRate            *bool `json:"movie_rate"`
	MovieCollection      *bool `json:"movie_collection"`
	EpisodeScrobbleStart *bool `json:"episode_scrobble_start"`
	EpisodeScrobbleStop  *bool `json:"episode_scrobble_stop"`
	EpisodeRate          *bool `json:"episode_rate"`
	EpisodeCollection    *bool `json:"episode_collection"`
	ShowRate             *bool `json:"show_rate"`
	SeasonRate           *bool `json:"season_rate"`
}

// User represents an authenticated user with their configuration
type User struct {
	ID             string    `json:"id"`
	Username       string    `json:"username"`
	PlexUsername   string    `json:"plex_username,omitempty"`
	AccessToken    string    `json:"access_token"`
	RefreshToken   string    `json:"refresh_token"`
	TokenExpiresAt time.Time `json:"token_expires_at"`
	Config         Config    `json:"config"`

	// Store reference (not serialised)
	Store Store `json:"-"`
}

// uuid generates a random UUID
func uuid() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		slog.Error("Error generating UUID", "error", err)
		return "00000000-0000-0000-0000-000000000000"
	}
	return fmt.Sprintf("%x%x%x%x%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// NewUser creates a new user with default configuration
func NewUser(username, accessToken, refreshToken string, expiresIn, createdAt int64, store Store) User {
	return NewUserWithID(uuid(), username, accessToken, refreshToken, expiresIn, createdAt, store)
}

// NewUserWithID creates a new user with a specific ID
func NewUserWithID(id, username, accessToken, refreshToken string, expiresIn, createdAt int64, store Store) User {
	tokenExpiresAt := time.Unix(createdAt, 0).Add(time.Duration(expiresIn) * time.Second)
	user := User{
		ID:             id,
		Username:       username,
		AccessToken:    accessToken,
		RefreshToken:   refreshToken,
		TokenExpiresAt: tokenExpiresAt,
		Store:          store,
		// Config fields remain nil (unconfigured)
	}
	slog.Info("User initialised", "id", id, "username", username)
	user.Save()
	return user
}

// UpdateUser updates authentication tokens
func (user *User) UpdateUser(accessToken, refreshToken string, expiresIn, createdAt int64) {
	user.AccessToken = accessToken
	user.RefreshToken = refreshToken
	user.TokenExpiresAt = time.Unix(createdAt, 0).Add(time.Duration(expiresIn) * time.Second)
	slog.Info("User token updated", "id", user.ID)
	user.Save()
}

// UpdateConfiguration updates the user's sync preferences
func (user *User) UpdateConfiguration(config Config, plexUsername string) {
	user.PlexUsername = plexUsername
	user.Config = config
	slog.Info("User configuration updated", "id", user.ID)
	user.Save()
}

// IsConfigured returns true if the user has complete configuration saved.
// All core settings must be present (not nil) for a valid configuration.
func (user User) IsConfigured() bool {
	c := user.Config
	return c.MovieScrobbleStart != nil &&
		c.MovieScrobbleStop != nil &&
		c.MovieRate != nil &&
		c.EpisodeScrobbleStart != nil &&
		c.EpisodeScrobbleStop != nil &&
		c.EpisodeRate != nil
}

// Save writes the user to the store
func (user *User) Save() error {
	if user.Store == nil {
		return fmt.Errorf("store is nil in User.Save()")
	}
	if err := user.Store.WriteUser(*user); err != nil {
		slog.Error("Error saving user", "id", user.ID, "error", err)
		return err
	}
	return nil
}

// Config getters with smart defaults for UI

func (c Config) GetMovieScrobbleStart() bool {
	if c.MovieScrobbleStart == nil {
		return true // Default: enabled
	}
	return *c.MovieScrobbleStart
}

func (c Config) GetMovieScrobbleStop() bool {
	if c.MovieScrobbleStop == nil {
		return true
	}
	return *c.MovieScrobbleStop
}

func (c Config) GetMovieRate() bool {
	if c.MovieRate == nil {
		return true
	}
	return *c.MovieRate
}

func (c Config) GetMovieCollection() bool {
	if c.MovieCollection == nil {
		return false // Default: disabled
	}
	return *c.MovieCollection
}

func (c Config) GetEpisodeScrobbleStart() bool {
	if c.EpisodeScrobbleStart == nil {
		return true
	}
	return *c.EpisodeScrobbleStart
}

func (c Config) GetEpisodeScrobbleStop() bool {
	if c.EpisodeScrobbleStop == nil {
		return true
	}
	return *c.EpisodeScrobbleStop
}

func (c Config) GetEpisodeRate() bool {
	if c.EpisodeRate == nil {
		return true
	}
	return *c.EpisodeRate
}

func (c Config) GetEpisodeCollection() bool {
	if c.EpisodeCollection == nil {
		return false // Default: disabled
	}
	return *c.EpisodeCollection
}

func (c Config) GetShowRate() bool {
	if c.ShowRate == nil {
		return true
	}
	return *c.ShowRate
}

func (c Config) GetSeasonRate() bool {
	if c.SeasonRate == nil {
		return true
	}
	return *c.SeasonRate
}
