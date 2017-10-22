package handlers

type MockDriver struct {
}

func (m *MockDriver) InsertUser(userID, username string) {}

func (m *MockDriver) GetRedditOAuthToken(userID string) (string, error) { return "", nil }

func (m *MockDriver) UpdateRedditAccount(userID, redditUser, authToken, tokenExpiry string) bool {
	return true
}

func (m *MockDriver) UpdateOAuthToken(userID, token, expiry string) bool { return true }
