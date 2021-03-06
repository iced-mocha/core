package storage

import (
	"github.com/iced-mocha/shared/models"
)

type Driver interface {
	InsertUser(user models.User) error

	GetUser(username string) (models.User, bool, error)

	GetRedditOAuthToken(userID string) (string, error)

	UpdateWeights(username string, weights models.Weights) bool

	UpdateRssFeeds(username string, feeds map[string][]string) error

	UpdateRedditAccount(userID, redditUser, authToken, refreshToken string) bool

	UpdateTwitterAccount(userID, twitterUser, authToken, secret string) bool

	UpdateFacebookAccount(userID, facebookUser, authToken string) bool

	UpdateOAuthToken(userID, token, expiry string) bool
}
