package handlers

import (
	"net/http"
)

type CoreAPI interface {
	GetPosts(w http.ResponseWriter, r *http.Request)
	InsertUser(w http.ResponseWriter, r *http.Request)
	GetUser(w http.ResponseWriter, r *http.Request)
	UpdateRssFeeds(w http.ResponseWriter, r *http.Request)
	Login(w http.ResponseWriter, r *http.Request)
	Logout(w http.ResponseWriter, r *http.Request)
	IsLoggedIn(w http.ResponseWriter, r *http.Request)
	TwitterAuth(w http.ResponseWriter, r *http.Request)
	RedditAuth(w http.ResponseWriter, r *http.Request)
	UpdateWeights(w http.ResponseWriter, r *http.Request)
	UpdateAccountAuth(w http.ResponseWriter, r *http.Request)
	DeleteLinkedAccount(w http.ResponseWriter, r *http.Request)
}
