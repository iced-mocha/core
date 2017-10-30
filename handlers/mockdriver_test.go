package handlers

import (
	"github.com/iced-mocha/shared/models"
)

type MockDriver struct {
}

func (m *MockDriver) InsertUser(user models.User) error { return nil }

func (m *MockDriver) GetUser(username string) (models.User, bool, error) {
	return models.User{}, true, nil
}

func (m *MockDriver) GetRedditOAuthToken(userID string) (string, error) { return "", nil }

func (m *MockDriver) UpdateRedditAccount(userID, redditUser, authToken, tokenExpiry string) bool {
	return true
}

func (m *MockDriver) UpdateOAuthToken(userID, token, expiry string) bool { return true }
