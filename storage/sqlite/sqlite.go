package sqlite

import (
	"database/sql"
	"errors"
	"log"
	"reflect"

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

type NullString sql.NullString

// Scan implements the Scanner interface for NullString
func (ns *NullString) Scan(value interface{}) error {
	var s sql.NullString
	if err := s.Scan(value); err != nil {
		return err
	}

	// if nil then make Valid false
	if reflect.TypeOf(value) == nil {
		*ns = NullString{s.String, false}
	} else {
		*ns = NullString{s.String, true}
	}

	return nil
}

// Inserts a user into the database
// NOTE: This assumes the password of the user object has already been hashed
func (d *driver) InsertUser(user models.User) error {
	log.Printf("Inserting user with ID: %v, and username: %v", user.ID, user.Username)

	stmt, err := d.db.Prepare("INSERT INTO UserInfo(UserID, Username, Password) values(?,?,?)")
	if err != nil {
		log.Printf("Unable to prepare statement for inserting user: %v", err)
		return err
	}

	_, err = stmt.Exec(user.ID, user.Username, user.Password)
	if err != nil {
		log.Println(err)
		return err
	}

	log.Printf("Successfuly inserted user with ID: %v, and username: %v", user.ID, user.Username)
	return nil
}

func (d *driver) GetRedditOAuthToken(username string) (string, error) {
	log.Printf("Attempting to get reddit token for user: %v\n", username)

	stmt, err := d.db.Prepare("SELECT RedditAuthToken FROM UserInfo WHERE Username=?")
	if err != nil {
		log.Println(err)
		return "", err
	}

	rows, err := stmt.Query(username)
	if err != nil {
		log.Println(err)
		return "", err
	}
	// This is need to prevent database locking
	defer rows.Close()

	// Try to get the first and hopefully only result from the query
	if !rows.Next() {
		log.Printf("Could not find user in DB: %v\n", username)
		return "", errors.New("No user found in database with given username: " + username)
	}

	var RedditAuthToken string
	rows.Scan(&RedditAuthToken)

	log.Printf("Successfully got auth token: %v for user %v.", RedditAuthToken, username)
	return RedditAuthToken, nil
}

// Attempts to get the user with the given username from the database
// Returns the user (if any), whether or not that user exists (bool) and a potential error
func (d *driver) GetUser(username string) (models.User, bool, error) {
	log.Printf("Attempting to retrieve user with username %v from db", username)
	var user models.User = models.User{}

	// This query will have a result set of 0 or 1.. 1 means Username already exists
	stmt, err := d.db.Prepare("SELECT * FROM UserInfo WHERE Username=?")
	if err != nil {
		log.Printf("Error preparing statement: %v", err)
		return user, false, err
	}

	rows, err := stmt.Query(username)
	if err != nil {
		log.Printf("Error completing query: %v", err)
		return user, false, err
	}
	// This is need to prevent database locking
	defer rows.Close()

	// If there is no rows.Next() Username does not exist
	if !rows.Next() {
		log.Printf("User %v not found in db", username)
		return user, false, nil
	}

	// Scan the select Row into our user struct
	// NOTE: It is import that this is kept up to date with database schema
	var redditUsername, redditAuthToken, facebookUsername, facebookAuthToken NullString
	err = rows.Scan(&user.ID, &user.Username, &user.Password, &redditUsername, &redditAuthToken, &facebookUsername, &facebookAuthToken)
	if err != nil {
		log.Printf("Unable to get users: %v", err)
		return user, false, err
	}

	user.RedditUsername = redditUsername.String
	user.RedditAuthToken = redditAuthToken.String
	user.FacebookUsername = facebookUsername.String
	user.FacebookAuthToken = facebookAuthToken.String

	// Otherwise the username does exist
	return user, true, nil

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
func (d *driver) UpdateRedditAccount(username, redditUser, authToken string) bool {
	query := "UPDATE UserInfo SET RedditUserName=?, RedditAuthToken=? where Username=?"

	stmt, err := d.db.Prepare(query)
	if err != nil {
		log.Printf("Unable to prepare query to update reddit account: %v", err)
		return false
	}

	res, err := stmt.Exec(redditUser, authToken, username)
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

// Updates information about a facebook account for a given userID
func (d *driver) UpdateFacebookAccount(username, facebookUser, authToken string) bool {
	query := "UPDATE UserInfo SET FacebookUserName=?, FacebookAuthToken=? where Username=?"

	stmt, err := d.db.Prepare(query)
	if err != nil {
		log.Printf("Unable to prepare query to update facebook account: %v", err)
		return false
	}

	res, err := stmt.Exec(facebookUser, authToken, username)
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
// TODO: Rename function or generalize
func (d *driver) UpdateOAuthToken(username, token, expiry string) bool {
	log.Printf("Going to update oauth token for user: %v", username)

	// TODO: If no records are updated we are not logging an error/message
	stmt, err := d.db.Prepare("UPDATE UserInfo SET RedditAuthToken=?, RedditTokenExpiry=? where Username=?")
	if err != nil {
		log.Println(err)
		return false
	}

	res, err := stmt.Exec(token, expiry, username)
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
	if config.DatabaseFile == "" {
		filename = databaseFile
	} else {
		filename = config.DatabaseFile
	}

	db, err := sql.Open(databaseDriver, filename)
	if err != nil {
		return nil, err
	}

	// Construct new driver object with database file name
	d := &driver{db}

	return d, nil
}
