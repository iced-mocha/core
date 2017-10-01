package main

import (
	"github.com/iced-mocha/core/handlers"
	"github.com/iced-mocha/core/server"
	"log"
	"net/http"
)

func main() {

	handler := &handlers.CoreHandler{}
	s, err := server.New(handler)
	if err != nil {
		log.Fatal("error initializing server: ", err)
	}

	log.Fatal(http.ListenAndServe(":3000", s.Router))
}
