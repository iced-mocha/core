package handlers

import (
	"github.com/iced-mocha/shared/models"
)

type MockDriver struct {
}

func (m *MockDriver) InsertUser(user models.User) error { return nil }

func (m *MockDriver) GetUser(username string) (models.User, bool, error) {
	if username == "exists" {
		return models.User{}, true, nil
	}
	return models.User{}, false, nil
}

func (m *MockDriver) GetRedditOAuthToken(userID string) (string, error) { return "", nil }

func (m *MockDriver) UpdateRedditAccount(userID, redditUser, authToken, refresh string) bool {
	return true
}

func (m *MockDriver) UpdateTwitterAccount(userID, redditUser, authToken, secret string) bool {
	return true
}

func (m *MockDriver) UpdateFacebookAccount(userID, facebookUser, authToken string) bool {
	return true
}

func (m *MockDriver) UpdateWeights(username string, weights models.Weights) bool { return true }

func (m *MockDriver) UpdateOAuthToken(userID, token, expiry string) bool { return true }
