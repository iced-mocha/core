package main

import (
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/iced-mocha/core/config/yaml"
	"github.com/iced-mocha/core/handlers"
	"github.com/iced-mocha/core/server"
	"github.com/iced-mocha/core/sessions"
	_ "github.com/iced-mocha/core/sessions/memory"
	"github.com/iced-mocha/core/storage/sql"
	"github.com/patrickmn/go-cache"
)

const (
	certFile = "server.crt"
	keyFile  = "server.key"
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
	driver, err := sql.New(sql.Config{
		DatabasePath:   os.Getenv("DATABASE_PATH"),
		DatabaseDriver: os.Getenv("DATABASE_TYPE"),
	})

	if err != nil {
		log.Fatalf("Unable to create driver: %v", err)
	}

	// Create our sessions manager
	sm, err := sessions.NewManager("memory", "icedmochasecret", 3600)
	if err != nil {
		log.Fatalf("Unable to create session manager: %v", err)
	}

	// Create our cache
	c := cache.New(30*time.Minute, 45*time.Minute)

	// Create our handler
	handler, err := handlers.New(driver, *sm, config, c)
	if err != nil {
		log.Fatalf("Unable to create handler", err)
	}

	s, err := server.New(handler)
	if err != nil {
		log.Fatalf("error initializing server: %v", err)
	}

	srv := &http.Server{
		Addr:      ":3000",
		Handler:   s,
		TLSConfig: &tls.Config{},
	}

	// TODO: Server will silently fail if server.crt or server.key do not exists
	srv.ListenAndServeTLS("/etc/ssl/certs/core.crt", "/etc/ssl/private/core.key")
}

func checkExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}
