package driver

import (
	"github.com/iced-mocha/shared/models"
)

type StorageDriver interface {
	InsertUser(user models.User) error

	GetUser(username string) (models.User, bool, error)

	GetRedditOAuthToken(userID string) (string, error)

	UpdateRedditAccount(userID, redditUser, authToken, tokenExpiry string) bool

	UpdateOAuthToken(userID, token, expiry string) bool
}
