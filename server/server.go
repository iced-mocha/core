package server

import (
	"github.com/gorilla/mux"
	"github.com/iced-mocha/core/handlers"
)

type Server struct {
	Router *mux.Router
}

func New(api handlers.CoreAPI) (*Server, error) {
	s := &Server{Router: mux.NewRouter()}

	s.Router.HandleFunc("/v1/posts", api.GetPosts).Methods("GET")

	// For now lets have core generate a user id and return in respsone body
	s.Router.HandleFunc("/v1/users", api.InsertUser).Methods("PUT")

	s.Router.HandleFunc("/v1/users/{userID}/authorize/reddit", api.RedditAuth).Methods("GET")
	s.Router.HandleFunc("/v1/users/{userID}/authorize/reddit", api.UpdateRedditAuth).Methods("POST")

	return s, nil
}
