package server

import (
	"github.com/gorilla/mux"
	"github.com/icedmocha/core/handlers"
)

type Server struct {
	Router *mux.Router
}

func New(api handlers.CoreAPI) (*Server, error) {
	s := &Server{Router: mux.NewRouter()}

	s.Router.HandleFunc("/v1/{id}/posts", api.GetPosts).Methods("GET")

	return s, nil
}
