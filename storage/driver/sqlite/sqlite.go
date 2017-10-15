package sqlite

import (
	"errors"
	"log"

	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	_ "github.com/twinj/uuid"
)

const (
	databaseFile   = "./database.db"
	databaseDriver = "sqlite3"
)

type driver struct {
	db *sql.DB
}

// Inserts a user into the database and creates an id for that user
func (d *driver) InsertUser(userID, username string) {
	// Generate a unique userid
	//u := uuid.NewV4()

	log.Printf("Inserting user with ID: %v, and username: %v", userID, username)

	stmt, err := d.db.Prepare("INSERT INTO UserInfo(UserID, Username) values(?,?)")
	if err != nil {
		log.Println(err)
		return
	}

	_, err = stmt.Exec(userID, username)
	if err != nil {
		log.Println(err)
		return
	}

	log.Printf("Successfuly inserted user with ID: %v, and username: %v", userID, username)
}

// TODO: This funciton isnt working
func (d *driver) GetRedditOAuthToken(userID string) (string, error) {
	log.Printf("Attempting to get token for user: %v\n", userID)

	stmt, err := d.db.Prepare("SELECT RedditAuthToken FROM UserInfo WHERE UserID=?")
	if err != nil {
		log.Println(err)
		return "", err
	}

	rows, err := stmt.Query(userID)
	if err != nil {
		log.Println(err)
		return "", err
	}

	// Try to get the first and hopefully only result from the query
	if !rows.Next() {
		log.Printf("Could not find user in DB: %v\n", userID)
		return "", errors.New("No user found in database with given id" + userID)
	}

	var RedditAuthToken string
	rows.Scan(&RedditAuthToken)

	log.Printf("Successfully got auth token for user")
	return RedditAuthToken, nil
}

func (d *driver) UpdateRedditAccount(userID, redditUser, authToken, tokenExpiry string) bool {
	noAuth := false
	query := "UPDATE UserInfo SET RedditUserName=?, RedditAuthToken=?, TokenExpiry=? where UserID=?"
	noAuthQuery := "UPDATE UserInfo SET RedditUserName=? WHERE UserID=?"

	// Decide which query were using
	if authToken == "" && tokenExpiry == "" {
		noAuth = true
		query = noAuthQuery
	}

	stmt, err := d.db.Prepare(query)
	if err != nil {
		log.Println(err)
		return false
	}

	var res sql.Result

	// Determine from before which is the proper statement to execute
	if noAuth {
		res, err = stmt.Exec(redditUser, userID)
	} else {
		res, err = stmt.Exec(redditUser, authToken, tokenExpiry, userID)
	}

	if err != nil {
		log.Println(err)
		return false
	}

	n, err := res.RowsAffected()
	if err != nil {
		log.Println(err)
		return false
	}

	// If the number of rows is greater than 0, then we have updated a user
	return n > 0

}

// Updates the auth token stored in db for the given userID
// Returns whether or not update was successful
func (d *driver) UpdateOAuthToken(userID, token, expiry string) bool {
	log.Printf("Going to update oauth token for user: %v", userID)

	// TODO: This needs to be changed -- just dont want to do it now
	// Dirty hack
	// Also this produces an error if the user already exists :-(
	d.InsertUser(userID, "iced-mocha")

	stmt, err := d.db.Prepare("UPDATE UserInfo SET RedditAuthToken=?, TokenExpiry=? where UserID=?")
	if err != nil {
		log.Println(err)
		return false
	}

	res, err := stmt.Exec(token, expiry, userID)
	if err != nil {
		log.Println(err)
		return false
	}

	n, err := res.RowsAffected()
	if err != nil {
		log.Println(err)
		return false
	}

	// If the number of rows is greater than 0, then we have updated a user
	return n > 0
}

// Creates a new driver containing pointer to sqlite db object
func New(config Config) (*driver, error) {
	var filename string

	// If we are provided a database file via the config object use it. Otherwise default
	if config.databaseFile == "" {
		filename = databaseFile
	} else {
		filename = config.databaseFile
	}

	db, err := sql.Open(databaseDriver, filename)
	if err != nil {
		return nil, err
	}

	// Construct new driver object with database file name
	d := &driver{db}

	return d, nil
}
