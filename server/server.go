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

	return s, nil
}
