package server

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/iced-mocha/core/handlers"
)

type Server struct {
	Router *mux.Router
}

func New(api handlers.CoreAPI) (*Server, error) {
	s := &Server{Router: mux.NewRouter()}

	s.Router.HandleFunc("/v1/posts", api.GetPosts).Methods("GET")
	s.Router.HandleFunc("/v1/users", api.InsertUser).Methods("POST")

	// Uses session id in cookie to retrieve user id
	s.Router.HandleFunc("/v1/users", api.GetUser).Methods("GET")
	s.Router.HandleFunc("/v1/login", api.Login).Methods("POST")
	s.Router.HandleFunc("/v1/logout", api.Logout).Methods("POST")
	s.Router.HandleFunc("/v1/loggedin", api.IsLoggedIn).Methods("GET")

	s.Router.HandleFunc("/v1/users/accounts/{type}", api.DeleteLinkedAccount).Methods("DELETE")

	s.Router.HandleFunc("/v1/users/{userID}/authorize/reddit", api.RedditAuth).Methods("GET")
	s.Router.HandleFunc("/v1/users/{userID}/authorize/reddit", api.UpdateRedditAuth).Methods("POST")

	return s, nil
}

func (s *Server) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if origin := req.Header.Get("Origin"); origin != "" {
		rw.Header().Set("Access-Control-Allow-Origin", origin)
		rw.Header().Set("Access-Control-Allow-Credentials", "true")
		rw.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		rw.Header().Set("Access-Control-Allow-Headers",
			"Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	}
	// Stop here if its Preflighted OPTIONS request
	if req.Method == "OPTIONS" {
		return
	}
	// Lets Gorilla work
	s.Router.ServeHTTP(rw, req)
}
