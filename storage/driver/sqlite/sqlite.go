package sqlite

import (
//"database/sql"
//test "github.com/mattn/go-sqlite3"
)

const (
	databaseFile   = "./database.db"
	databaseDriver = "sqlite3"
)

type driver struct {
	databaseFile string
}

func New(config Config) (*driver, error) {
	var filename string

	// If we are provided a database file via the config object use it. Otherwise default
	if config.databaseFile == "" {
		filename = databaseFile
	} else {
		filename = config.databaseFile
	}

	// Construct new driver object with database file name
	d := &driver{filename}

	return d, nil
}
