package driver

type StorageDriver interface {
	InsertUser(userID, username string)

	GetRedditOAuthToken(userID string) (string, error)

	UpdateRedditAccount(userID, redditUser, authToken, tokenExpiry string) bool

	UpdateOAuthToken(userID, token, expiry string) bool
}
