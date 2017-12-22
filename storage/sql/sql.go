package sql

import (
	"database/sql"
	"errors"
	"log"
	"reflect"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/iced-mocha/shared/models"
	_ "github.com/mattn/go-sqlite3"
	_ "github.com/twinj/uuid"
)

const (
	databasePath   = "./database.db"
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
	var err error
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}()

	pw := user.PostWeights
	_, err = tx.Exec(`
		INSERT INTO UserInfo (
			UserID, Username, Password,
			RedditWeight, FacebookWeight, HackerNewsWeight, GoogleNewsWeight, TwitterWeight
		)
		VALUES (?,?,?,?,?,?,?,?)`,
		user.ID, user.Username, user.Password,
		pw.Reddit, pw.Facebook, pw.HackerNews, pw.GoogleNews, pw.Twitter)
	if err != nil {
		log.Println(err)
		return err
	}

	values := []string{}
	args := make([]interface{}, 0)
	for name, group := range user.RssGroups {
		values = append(values, "(?,?,?,?)")
		args = append(args, user.Username, strings.Join(group, ","), user.PostWeights.RSS[name], name)
	}

	_, err = tx.Exec(`
		INSERT INTO Rss (Username, Feeds, Weight, Name)
		VALUES `+strings.Join(values, ","), args...)
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

func (d *driver) GetTwitterSecrets(username string) (string, string, error) {
	log.Printf("Attempting to get twitter secrets for user: %v", username)

	stmt, err := d.db.Prepare("SELECT TwitterAuthToken, TwitterSecret FROM UserInfo WHERE Username=?")
	if err != nil {
		log.Println(err)
		return "", "", err
	}

	rows, err := stmt.Query(username)
	if err != nil {
		log.Println(err)
		return "", "", err
	}
	// This is need to prevent database locking
	defer rows.Close()

	// Try to get the first and hopefully only result from the query
	if !rows.Next() {
		log.Printf("Could not find user in DB: %v\n", username)
		return "", "", errors.New("No user found in database with given username: " + username)
	}

	var TwitterAuthToken, TwitterSecret string
	rows.Scan(&TwitterAuthToken, &TwitterSecret)

	log.Printf("Successfully twitter token and secret for user %v.", username)
	return TwitterAuthToken, TwitterSecret, nil
}

// Attempts to get the user with the given username from the database
// Returns the user (if any), whether or not that user exists (bool) and a potential error
func (d *driver) GetUser(username string) (models.User, bool, error) {
	log.Printf("Attempting to retrieve user with username %v from db", username)
	var user models.User = models.User{
		Username:  username,
		RssGroups: make(map[string][]string),
		PostWeights: models.Weights{
			RSS: make(map[string]float64),
		},
	}

	rows, err := d.db.Query(`
		SELECT UserID, UserInfo.Username, Password, TwitterUsername, TwitterAuthToken, TwitterSecret,
			RedditUsername, RedditAuthToken, RedditRefreshToken, FacebookUsername, FacebookAuthToken,
			RedditWeight, FacebookWeight, HackerNewsWeight, GoogleNewsWeight, TwitterWeight,
			Rss.Feeds, Rss.Weight, Rss.Name
		FROM UserInfo
		LEFT JOIN Rss ON UserInfo.Username=Rss.Username
		WHERE UserInfo.Username=?
	`, username)
	if err != nil {
		log.Printf("Error completing query: %v", err)
		return user, false, err
	}
	// This is need to prevent database locking
	defer rows.Close()

	var foundUser bool
	for rows.Next() {
		foundUser = true
		var rssFeeds sql.NullString
		var rssWeight sql.NullFloat64
		var rssName sql.NullString
		rows.Scan(
			&user.ID,
			&user.Username,
			&user.Password,
			&user.TwitterUsername,
			&user.TwitterAuthToken,
			&user.TwitterSecret,
			&user.RedditUsername,
			&user.RedditAuthToken,
			&user.RedditRefreshToken,
			&user.FacebookUsername,
			&user.FacebookAuthToken,
			&user.PostWeights.Reddit,
			&user.PostWeights.Facebook,
			&user.PostWeights.HackerNews,
			&user.PostWeights.GoogleNews,
			&user.PostWeights.Twitter,
			&rssFeeds, &rssWeight, &rssName,
		)

		if rssName.Valid {
			name := rssName.String
			user.RssGroups[name] = strings.Split(rssFeeds.String, ",")
			user.PostWeights.RSS[name] = rssWeight.Float64
		}
	}

	if !foundUser {
		log.Printf("User %v not found in db", username)
		return user, false, nil
	}

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

func (d *driver) UpdateWeights(username string, weights models.Weights) bool {
	log.Printf("Preparing to insert weights into db for user: %v", username)
	var err error
	tx, err := d.db.Begin()
	if err != nil {
		log.Println(err)
		return false
	}
	defer func() {
		if err == nil {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}()

	res, err := tx.Exec(`
		UPDATE UserInfo SET RedditWeight=?, FacebookWeight=?, HackerNewsWeight=?, GoogleNewsWeight=?, TwitterWeight=? WHERE Username=?
	`, weights.Reddit, weights.Facebook, weights.HackerNews, weights.GoogleNews, weights.Twitter, username)
	if err != nil {
		log.Println(err)
		return false
	}

	n, err := res.RowsAffected()
	if err != nil {
		log.Println(err)
		return false
	}
	if n == 0 {
		log.Println("Could not find user %v when updating weights", username)
		return false
	}

	for name, weight := range weights.RSS {
		res, err = tx.Exec(`
			UPDATE Rss SET Weight=? WHERE Username=? AND Name=?
		`, weight, username, name)
		if err != nil {
			log.Println(err)
			return false
		}
	}

	return true
}

// Updates information about a reddit account for a given userID
func (d *driver) UpdateRedditAccount(username, redditUser, authToken, refreshToken string) bool {
	query := "UPDATE UserInfo SET RedditUserName=?, RedditAuthToken=?, RedditRefreshToken=? where Username=?"

	stmt, err := d.db.Prepare(query)
	if err != nil {
		log.Printf("Unable to prepare query to update reddit account: %v", err)
		return false
	}

	res, err := stmt.Exec(redditUser, authToken, refreshToken, username)
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

// Updates information about a reddit account for a given userID
func (d *driver) UpdateTwitterAccount(username, twitterUser, authToken, secret string) bool {
	query := "UPDATE UserInfo SET TwitterUserName=?, TwitterAuthToken=?, TwitterSecret=? where Username=?"

	stmt, err := d.db.Prepare(query)
	if err != nil {
		log.Printf("Unable to prepare query to update twitter account: %v", err)
		return false
	}

	res, err := stmt.Exec(twitterUser, authToken, secret, username)
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

func (d *driver) UpdateRssFeeds(username string, feeds map[string][]string) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}()

	caseArgs := make([]interface{}, 0)
	caseVals := []string{}
	whereArgs := make([]interface{}, 0)
	whereVals := []string{}
	for name, feeds := range feeds {
		caseArgs = append(caseArgs, username, name, strings.Join(feeds, ","))
		caseVals = append(caseVals, "WHEN ? || ? THEN ?")
		whereArgs = append(whereArgs, username, name)
		whereVals = append(whereVals, "? || ?")
	}
	_, err = tx.Exec(`
		UPDATE Rss SET Feeds = CASE Username || Name
			`+strings.Join(caseVals, " ")+`
			ELSE Feeds
			END
		WHERE Username || Name IN (`+strings.Join(whereVals, ",")+`)
	`, append(caseArgs, whereArgs...)...)
	if err != nil {
		return err
	}

	values := []string{}
	args := make([]interface{}, 0)
	for name := range feeds {
		values = append(values, "?")
		args = append(args, name)
	}

	_, err = tx.Exec(`
		DELETE FROM Rss WHERE NOT Name IN (`+strings.Join(values, ",")+`) AND Username=?
	`, append(args, username)...)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

// Creates a new driver containing pointer to sqlite db object
func New(config Config) (*driver, error) {
	var dbPath string
	var dbDriver string

	// If we are provided a database file via the config object use it. Otherwise default
	if config.DatabasePath == "" {
		dbPath = databasePath
	} else {
		dbPath = config.DatabasePath
	}

	if config.DatabaseDriver == "" {
		dbDriver = databaseDriver
	} else {
		dbDriver = config.DatabaseDriver
	}

	db, err := sql.Open(dbDriver, dbPath)
	if err != nil {
		return nil, err
	}

	// Construct new driver object with database file name
	d := &driver{db}

	return d, nil
}
