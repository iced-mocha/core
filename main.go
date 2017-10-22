package main

import (
	"log"
	"net/http"

	"github.com/iced-mocha/core/handlers"
	"github.com/iced-mocha/core/server"
	"github.com/iced-mocha/core/storage/driver/sqlite"
)

func main() {
	// Create our storage driver
	driver, err := sqlite.New(sqlite.Config{})
	if err != nil {
		log.Fatalf("Unable to create driver: %v\n", err)
	}

	s, err := server.New(&handlers.CoreHandler{driver})
	if err != nil {
		log.Fatalf("error initializing server: %v", err)
	}

	log.Fatal(http.ListenAndServe(":3000", s.Router))
}
