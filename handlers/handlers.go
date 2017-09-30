package handlers

import (
	"net/http"
)

type CoreHandler struct{}

func (api *CoreHandler) GetPosts(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}
