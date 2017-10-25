package sqlite

import (
	"database/sql"
	"errors"
	"log"

	"github.com/iced-mocha/shared/models"
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
func (d *driver) InsertUser(user models.User) error {
	log.Printf("Inserting user with ID: %v, and username: %v", user.ID, user.Username)

	stmt, err := d.db.Prepare("INSERT INTO UserInfo(UserID, Username, RedditUserName) values(?,?,?)")
	if err != nil {
		log.Println(err)
		return err
	}

	_, err = stmt.Exec(user.ID, user.Username, user.RedditUser)
	if err != nil {
		log.Println(err)
		return err
	}

	log.Printf("Successfuly inserted user with ID: %v, and username: %v", user.ID, user.Username)
	return nil
}

// TODO: This funciton isnt working -- why do i write this and not say how its not working
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
	// This is need to prevent database locking
	defer rows.Close()

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

// This function consumes a user name and sees if it already exists in the database
func (d *driver) UsernameExists(username string) (bool, error) {
	// This query will have a result set of 0 or 1.. 1 means Username already exists
	stmt, err := d.db.Prepare("SELECT 1 FROM UserInfo WHERE Username=?")
	if err != nil {
		log.Println(err)
		return false, err
	}

	rows, err := stmt.Query(username)
	if err != nil {
		log.Println(err)
		return false, err
	}
	// This is need to prevent database locking
	defer rows.Close()

	// If there is no rows.Next() Username is available to use
	if !rows.Next() {
		return false, nil
	}

	// Otherwise the username does exist
	return true, nil
}

// Updates information about a reddit account for a given userID
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

	// TODO -- to fix this we need a sign up button on front-end to trigger an insert user
	// Then after words logging in via reddit will work
	//d.InsertUser(userID, "iced-mocha")

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
