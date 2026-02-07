package store

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	keystorePath = "keystore"
	indexFile    = "usernames.json"
)

// DiskStore is a storage backend using local filesystem with JSON files
type DiskStore struct {
	basePath string
	mu       sync.RWMutex
}

// NewDiskStore creates a new disk-based storage
func NewDiskStore() *DiskStore {
	// Ensure keystore directory exists
	if err := os.MkdirAll(keystorePath, 0755); err != nil {
		slog.Error("Failed to create keystore directory", "error", err)
	}
	return &DiskStore{basePath: keystorePath}
}

// Ping verifies the storage is accessible
func (s *DiskStore) Ping() error {
	_, err := os.Stat(s.basePath)
	return err
}

// WriteUser saves a user to disk as a single JSON file
func (s *DiskStore) WriteUser(user User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Marshal user to JSON (Store field excluded via json:"-")
	data, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal user: %w", err)
	}

	// Write user file atomically
	userPath := filepath.Join(s.basePath, user.ID+".json")
	if err := s.atomicWrite(userPath, data); err != nil {
		return fmt.Errorf("failed to write user file: %w", err)
	}

	// Update username index
	if user.Username != "" {
		if err := s.updateIndex(user.Username, user.ID); err != nil {
			slog.Warn("Failed to update username index", "error", err)
		}
	}

	return nil
}

// GetUser loads a user by ID
func (s *DiskStore) GetUser(id string) *User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userPath := filepath.Join(s.basePath, id+".json")
	data, err := os.ReadFile(userPath)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Debug("Failed to read user file", "id", id, "error", err)
		}
		return nil
	}

	var user User
	if err := json.Unmarshal(data, &user); err != nil {
		slog.Error("Failed to unmarshal user", "id", id, "error", err)
		return nil
	}

	user.Store = s
	return &user
}

// GetUserByUsername looks up a user by their username
func (s *DiskStore) GetUserByUsername(username string) *User {
	s.mu.RLock()
	index := s.loadIndex()
	s.mu.RUnlock()

	id, ok := index[strings.ToLower(username)]
	if !ok {
		return nil
	}
	return s.GetUser(id)
}

// DeleteUser removes a user and their index entry
func (s *DiskStore) DeleteUser(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get user first to remove from index
	userPath := filepath.Join(s.basePath, id+".json")
	data, err := os.ReadFile(userPath)
	if err == nil {
		var user User
		if json.Unmarshal(data, &user) == nil && user.Username != "" {
			s.removeFromIndex(user.Username)
		}
	}

	// Delete user file
	if err := os.Remove(userPath); err != nil && !os.IsNotExist(err) {
		slog.Error("Failed to delete user file", "id", id, "error", err)
		return false
	}

	return true
}

// atomicWrite writes data to a file atomically using a temp file
func (s *DiskStore) atomicWrite(path string, data []byte) error {
	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

// loadIndex reads the username -> ID mapping
func (s *DiskStore) loadIndex() map[string]string {
	indexPath := filepath.Join(s.basePath, indexFile)
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return make(map[string]string)
	}

	var index map[string]string
	if err := json.Unmarshal(data, &index); err != nil {
		slog.Warn("Failed to parse username index", "error", err)
		return make(map[string]string)
	}
	return index
}

// updateIndex adds or updates a username -> ID mapping
func (s *DiskStore) updateIndex(username, id string) error {
	index := s.loadIndex()
	index[strings.ToLower(username)] = id

	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}

	indexPath := filepath.Join(s.basePath, indexFile)
	return s.atomicWrite(indexPath, data)
}

// removeFromIndex removes a username from the index
func (s *DiskStore) removeFromIndex(username string) {
	index := s.loadIndex()
	delete(index, strings.ToLower(username))

	data, _ := json.MarshalIndent(index, "", "  ")
	indexPath := filepath.Join(s.basePath, indexFile)
	s.atomicWrite(indexPath, data)
}
