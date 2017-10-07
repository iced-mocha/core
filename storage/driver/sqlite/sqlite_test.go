package sqlite

import (
	"log"
	"testing"
)

func TestNew(t *testing.T) {
	// Creating a basic driver should work so long as the file is there
	_, err := New(Config{})
	if err != nil {
		log.Fatalf("Unable to create driver: %v\n", err)
	}
}
