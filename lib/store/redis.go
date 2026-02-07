package store

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/redis/go-redis/v9"
)

// RedisStore is a storage backend using Redis
type RedisStore struct {
	client *redis.Client
	mu     sync.Mutex
}

// NewRedisClient creates a new Redis client
func NewRedisClient(addr, password string) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})

	ctx := context.Background()
	if _, err := client.Ping(ctx).Result(); err != nil {
		slog.Error("Redis connection failed", "error", err)
		os.Exit(1)
	}
	return client
}

// NewRedisStore creates a new Redis-backed store
func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client}
}

// Ping verifies Redis connectivity
func (s *RedisStore) Ping() error {
	return s.client.Ping(context.Background()).Err()
}

// WriteUser saves a user to Redis as JSON
func (s *RedisStore) WriteUser(user User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()
	key := "goplaxt:user:" + user.ID

	// Serialise user to JSON
	data, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("failed to marshal user: %w", err)
	}

	// Store user data
	if err := s.client.Set(ctx, key, data, 0).Err(); err != nil {
		return fmt.Errorf("failed to write user: %w", err)
	}

	// Update username index
	if user.Username != "" {
		indexKey := "goplaxt:username:" + strings.ToLower(user.Username)
		if err := s.client.Set(ctx, indexKey, user.ID, 0).Err(); err != nil {
			slog.Warn("Failed to update username index", "error", err)
		}
	}

	return nil
}

// GetUser loads a user by ID
func (s *RedisStore) GetUser(id string) *User {
	ctx := context.Background()
	key := "goplaxt:user:" + id

	data, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if err != redis.Nil {
			slog.Debug("Failed to get user", "id", id, "error", err)
		}
		return nil
	}

	var user User
	if err := json.Unmarshal([]byte(data), &user); err != nil {
		slog.Error("Failed to unmarshal user", "id", id, "error", err)
		return nil
	}

	user.Store = s
	return &user
}

// GetUserByUsername looks up a user by username
func (s *RedisStore) GetUserByUsername(username string) *User {
	ctx := context.Background()
	indexKey := "goplaxt:username:" + strings.ToLower(username)

	id, err := s.client.Get(ctx, indexKey).Result()
	if err != nil {
		if err != redis.Nil {
			slog.Debug("Username not found in index", "username", username)
		}
		return nil
	}

	return s.GetUser(id)
}

// DeleteUser removes a user and their index entry
func (s *RedisStore) DeleteUser(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()

	// Get user to remove from index
	if user := s.GetUser(id); user != nil && user.Username != "" {
		indexKey := "goplaxt:username:" + strings.ToLower(user.Username)
		s.client.Del(ctx, indexKey)
	}

	// Delete user data
	key := "goplaxt:user:" + id
	if err := s.client.Del(ctx, key).Err(); err != nil {
		slog.Error("Failed to delete user", "id", id, "error", err)
		return false
	}

	return true
}
