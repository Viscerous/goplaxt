package store

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestPostgresqlLoadingUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	mock.ExpectQuery(
		"SELECT username, access, refresh, expires_at FROM users WHERE id=.*",
	).WithArgs(
		"id123",
	).WillReturnRows(
		sqlmock.NewRows([]string{"username", "access", "refresh", "expires_at"}).
			AddRow(
				"halkeye",
				"access123",
				"refresh123",
				time.Date(2025, 03, 28, 22, 30, 55, 0, time.UTC),
			),
	)

	store := NewPostgresqlStore(db)

	expected, _ := json.Marshal(&User{
		ID:             "id123",
		Username:       "halkeye",
		AccessToken:    "access123",
		RefreshToken:   "refresh123",
		TokenExpiresAt: time.Date(2025, 03, 28, 22, 30, 55, 0, time.UTC),
		Store:          store,
	})
	actual, _ := json.Marshal(store.GetUser("id123"))

	assert.EqualValues(t, string(expected), string(actual))
}

func TestPostgresqlSavingUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	mock.ExpectExec("INSERT INTO ").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT").WithArgs("id123").WillReturnRows(
		sqlmock.NewRows([]string{"username", "access", "refresh", "expires_at"}).
			AddRow(
				"halkeye",
				"access123",
				"refresh123",
				time.Date(2025, 03, 28, 22, 30, 55, 0, time.UTC),
			),
	)

	store := NewPostgresqlStore(db)
	originalUser := &User{
		ID:             "id123",
		Username:       "halkeye",
		AccessToken:    "access123",
		RefreshToken:   "refresh123",
		TokenExpiresAt: time.Date(2025, 03, 28, 22, 30, 55, 0, time.UTC),
		Store:          store,
	}

	originalUser.save()

	expected, err := json.Marshal(originalUser)
	actual, err := json.Marshal(store.GetUser("id123"))

	assert.EqualValues(t, string(expected), string(actual))
}
