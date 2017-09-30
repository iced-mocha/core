package handlers

import (
	"net/http"
)

type CoreAPI interface {
	// Test endpoint
	GetPosts(w http.ResponseWriter, r *http.Request)
}
