package handlers

import (
	"net/http"
)

type CoreAPI interface {
	GetPosts(w http.ResponseWriter, r *http.Request)
	InsertUser(w http.ResponseWriter, r *http.Request)
	Login(w http.ResponseWriter, r *http.Request)
	RedditAuth(w http.ResponseWriter, r *http.Request)
	UpdateRedditAuth(w http.ResponseWriter, r *http.Request)
}
