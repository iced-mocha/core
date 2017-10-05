package sqlite

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"testing"
)

func TestNew(t *testing.T) {
	// Creating a basic driver should work so long as the file is there
	_, err := New(Config{})
	if err != nil {
		log.Fatalf("Unable to create driver: %v\n", err)
	}
	/*
		_, err := sql.Open("sqlite3", "./foo.db")
		if err != nil {
			log.Fatalf("Unable to open database: %v", err)
		}*/
}
