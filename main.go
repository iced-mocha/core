package main

import (
	"log"
	"net/http"
	"os"

	"github.com/iced-mocha/core/config/yaml"
	"github.com/iced-mocha/core/handlers"
	"github.com/iced-mocha/core/server"
	"github.com/iced-mocha/core/sessions"
	_ "github.com/iced-mocha/core/sessions/memory"
	"github.com/iced-mocha/core/storage/sqlite"
)

func main() {
	configFileName := "workspace.local.yml"

	// Create our config object, look for a file called workspace.local.yml
	// Fall back onto workspace.docker.yml (This is to allow quick development outside of docker)
	if _, err := os.Stat(configFileName); os.IsNotExist(err) {
		configFileName = "workspace.docker.yml"
	}

	// Create config object from the file
	config, err := yaml.New(configFileName)
	if err != nil {
		log.Fatalf("Unable to create configuration: %v", err)
	}

	// Create our storage driver
	driver, err := sqlite.New(sqlite.Config{})
	if err != nil {
		log.Fatalf("Unable to create driver: %v", err)
	}

	// Create our sessions manager
	sm, err := sessions.NewManager("memory", "icedmochasecret", 3600)
	if err != nil {
		log.Fatalf("Unable to create session manager: %v", err)
	}

	// Create our handler
	handler, err := handlers.New(driver, *sm, config)
	if err != nil {
		log.Fatalf("Unable to create handler", err)
	}

	s, err := server.New(handler)
	if err != nil {
		log.Fatalf("error initializing server: %v", err)
	}
	http.Handle("/", s)
	http.ListenAndServe(":3000", nil)
}
