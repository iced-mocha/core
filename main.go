package main

import (
	"log"
	"net/http"
	"os"

	"github.com/iced-mocha/core/config/yaml"
	"github.com/iced-mocha/core/handlers"
	"github.com/iced-mocha/core/server"
	"github.com/iced-mocha/core/storage/driver/sqlite"
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
		log.Fatalf("Unable to create configuration: %v\n", err)
	}

	// Create our storage driver
	driver, err := sqlite.New(sqlite.Config{})
	if err != nil {
		log.Fatalf("Unable to create driver: %v\n", err)
	}

	// Create our handler
	handler, err := handlers.New(driver, config)
	if err != nil {
		log.Fatalf("Unable to create handler\n")
	}

	s, err := server.New(handler)
	if err != nil {
		log.Fatalf("error initializing server: %v", err)
	}

	log.Fatal(http.ListenAndServe(":3000", s.Router))
}
