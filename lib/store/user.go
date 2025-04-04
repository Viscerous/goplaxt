package store

import (
	"fmt"
	"log"
	"os"
	"time"
)

type store interface {
	WriteUser(user User)
}

// User object
type User struct {
	ID             string
	Username       string
	AccessToken    string
	RefreshToken   string
	TokenExpiresAt time.Time
	Store          store
}

func uuid() string {
	f, _ := os.OpenFile("/dev/urandom", os.O_RDONLY, 0)
	b := make([]byte, 16)
	f.Read(b)
	f.Close()
	uuid := fmt.Sprintf("%x%x%x%x%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])

	return uuid
}

// NewUser creates a new user object
func NewUser(username, accessToken, refreshToken string, expiresIn int64, createdAt int64, store store) User {
	id := uuid()
	tokenExpiresAt := time.Unix(createdAt, 0).Add(time.Duration(expiresIn) * time.Second)
	user := User{
		ID:             id,
		Username:       username,
		AccessToken:    accessToken,
		RefreshToken:   refreshToken,
		TokenExpiresAt: tokenExpiresAt,
		Store:          store,
	}
	user.save()
	return user
}

// UpdateUser updates an existing user object
func (user *User) UpdateUser(accessToken, refreshToken string, expiresIn int64, createdAt int64) {
	user.AccessToken = accessToken
	user.RefreshToken = refreshToken
	user.TokenExpiresAt = time.Unix(createdAt, 0).Add(time.Duration(expiresIn) * time.Second)

	user.save()
}

func (user *User) save() {
	if user.Store == nil {
		log.Panic("Store is nil in User.save()")
	}
	log.Printf("Saving user: %+v", *user)
	user.Store.WriteUser(*user)
}
