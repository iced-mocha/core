package handlers

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/iced-mocha/core/clients"
	"github.com/iced-mocha/core/clients/facebook"
	"github.com/iced-mocha/core/clients/googlenews"
	"github.com/iced-mocha/core/clients/hackernews"
	"github.com/iced-mocha/core/clients/reddit"
	"github.com/iced-mocha/core/config"
	"github.com/iced-mocha/core/ranking"
	"github.com/iced-mocha/core/sessions"
	"github.com/iced-mocha/core/storage"
	"github.com/iced-mocha/shared/models"
	"github.com/patrickmn/go-cache"
	"github.com/satori/go.uuid"
	"golang.org/x/crypto/bcrypt"
)

type CoreHandler struct {
	Driver         storage.Driver
	Config         config.Config
	SessionManager sessions.Manager
	// TODO: Having a cache on core used for pagination requires us to only run
	// one instance of core
	Cache              *cache.Cache
	getNextPagingToken func() string

	Clients []clients.Client
}

// Structure received when updating provider auth info
type ProviderAuth struct {
	Type         string `json:"type"`
	Username     string `json:"username"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh-token"`
}

func New(d storage.Driver, sm sessions.Manager, conf config.Config, c *cache.Cache) (*CoreHandler, error) {
	handler := &CoreHandler{}
	handler.Driver = d
	handler.Config = conf
	handler.SessionManager = sm
	handler.Cache = c

	var idCounter int32
	handler.getNextPagingToken = func() string {
		idCounter++
		return strconv.FormatInt(int64(idCounter), 32)
	}

	// Start our session garbage collection
	//go handler.SessionManager.GC()

	// TODO Find a better way to do this
	// Maybe create a GetStringKeys function that returns array of values and a potential error

	hosts, err := handler.Config.GetStrings([]string{"hacker-news.host", "facebook.host", "reddit.host", "google-news.host"})
	if err != nil {
		return nil, err
	}

	ports, err := handler.Config.GetInts([]string{"hacker-news.port", "facebook.port", "reddit.port", "google-news.port"})
	if err != nil {
		return nil, err
	}

	handler.Clients = make([]clients.Client, 4)
	handler.Clients[0] = hackernews.New(hosts[0], ports[0], 2.0)
	handler.Clients[1] = facebook.New(hosts[1], ports[1], 1.0)
	handler.Clients[2] = reddit.New(hosts[2], ports[2], 2.0)
	handler.Clients[3] = googlenews.New(hosts[3], ports[3], 4.0)

	return handler, nil
}

// Deletes the type of account for authenticated user in the request
// DELETE /v1/users/accounts/{type}
func (h *CoreHandler) DeleteLinkedAccount(w http.ResponseWriter, r *http.Request) {
	s, err := h.SessionManager.GetSession(r)
	if err != nil {
		log.Printf("Could not get retrieve session for user: %v", err)
		// Return unauthorized error -- but TODO: in the future differentiate between 401 and 500
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Get user associate with the session
	ui := s.Get("username")
	username, ok := ui.(string)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Parse the type of account to remove
	t := mux.Vars(r)["type"]

	if t == "reddit" {
		// Overwriting all values with "" is essentially deleting
		h.Driver.UpdateRedditAccount(username, "", "")
	} else if t == "facebook" {
		h.Driver.UpdateFacebookAccount(username, "", "")
	}

	w.WriteHeader(http.StatusOK)
}

// Updates the facebook auth info stored for a user with id <userID>
// POST /v1/user/{userID}/authorize/facebook
func (handler *CoreHandler) UpdateFacebookAuth(w http.ResponseWriter, r *http.Request) {

	// TODO: We are currently not verifying that the user requesting this is in fact allowed to do so

	// Read body of the request
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}

	// Get the user id from path paramater
	id := mux.Vars(r)["userID"]

	log.Printf("About to update facebook auth information for user: %v", id)

	// Change the body into a user object
	auth := &ProviderAuth{}
	err = json.Unmarshal(body, auth)
	if err != nil {
		log.Printf("Error parsing request body when updating facebook auth for user: %v - %v", id, err)
		http.Error(w, "can't parse body", http.StatusBadRequest)
		return
	}

	successful := handler.Driver.UpdateFacebookAccount(id, auth.Username, auth.Token)
	if !successful {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Updates the reddit oauth token stored for a user with id <userID>
// POST /v1/user/{userID}/authorize/reddit
func (handler *CoreHandler) UpdateRedditAuth(w http.ResponseWriter, r *http.Request) {
	// TODO: We are currently not verifying that the user requesting this is in fact allowed to do so

	// Read body of the request
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}

	// Get the user id from path paramater
	id := mux.Vars(r)["userID"]

	// Change the body into a user object
	auth := &ProviderAuth{}
	err = json.Unmarshal(body, auth)
	if err != nil {
		log.Printf("Error parsing request body when updating reddit auth for user: %v - %v", id, err)
		http.Error(w, "can't parse body", http.StatusBadRequest)
		return
	}

	successful := handler.Driver.UpdateRedditAccount(id, auth.Username, auth.Token)
	if !successful {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Redirects to our reddit client to authorize or service to use reddit account
// GET /v1/user/{userID}/authorize/reddit
func (handler *CoreHandler) RedditAuth(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "http://reddit-client:3001/v1/authorize", http.StatusFound)
}

// Consumes plaintext password and hashes using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

// User for authenticating login to compare password and hash
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (handler *CoreHandler) IsLoggedIn(w http.ResponseWriter, r *http.Request) {
	// First check to see if the user is already logged in
	if handler.SessionManager.HasSession(r) {
		w.Write([]byte(`{ "logged-in": true }`))
		return
	}
	w.Write([]byte(`{ "logged-in": false }`))
}

func (handler *CoreHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// On logout all we need to do is destroy our cookies and session data
	log.Printf("Logging out user")
	handler.SessionManager.SessionDestroy(w, r)
	w.WriteHeader(http.StatusOK)
}

func buildJSONError(message string) string {
	return `{ "error": "` + message + `" }`
}

func (handler *CoreHandler) Login(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		http.Error(w, buildJSONError("bad request when attempting to login"), http.StatusBadRequest)
		return
	}

	// NOTE: We expect a user object but username and password are all that will be non-empty
	log.Printf("Received the following user to login: %v", string(body))

	// Marshal the body into a user object
	attemptedUser := &models.User{}
	if err := json.Unmarshal(body, attemptedUser); err != nil {
		log.Printf("Error parsing body: %v", err)
		http.Error(w, buildJSONError("bad request when attempting to login"), http.StatusBadRequest)
		return
	}

	// First check to see if the user is already logged in
	if handler.SessionManager.HasSession(r) {
		log.Printf("User %v attempted to log in, but they are already logged in.", attemptedUser.Username)
		// Not sure if this should be an error
		w.WriteHeader(http.StatusOK)
		return
	}

	// Get the actual user for the given username
	actualUser, exists, err := handler.Driver.GetUser(attemptedUser.Username)
	if err != nil {
		log.Printf("Unable to retrieve user: %v", err)
		http.Error(w, buildJSONError("internal server error when attempting to login"), http.StatusInternalServerError)
		return
	}

	// If the user does not exist return 401 (Unauthorized) for security reasons
	if !exists {
		log.Printf("Requested user %v does not exist", attemptedUser.Username)
		http.Error(w, buildJSONError("incorrect username or password"), http.StatusUnauthorized)
		return
	}

	// Otherwise the user exists so lets see if we were provided correct credentials
	// NOTE: attemptedUser.Password is plaintext and actualUser.Password is bcrypted hash of password
	valid := CheckPasswordHash(attemptedUser.Password, actualUser.Password)
	if !valid {
		// Not valid so return unauthorized
		log.Printf("Bad credentials attempting to authenticate user %v", attemptedUser.Username)
		http.Error(w, buildJSONError("incorrect username or password"), http.StatusUnauthorized)
		return
	}

	// Successfully logged in make sure we have a session -- will insert a session id into the ResponseWriters cookies
	session := handler.SessionManager.SessionStart(w, r)
	// Links the session id to our username
	session.Set("username", attemptedUser.Username)

	w.WriteHeader(http.StatusOK)
}

// Gets the user information tied to the session id in request
// GET /v1/users
func (handler *CoreHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	s, err := handler.SessionManager.GetSession(r)
	if err != nil {
		// Return unauthorized error -- but TODO: in the future differentiate between 401 and 500
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Get user associate with the session
	ui := s.Get("username")
	username, ok := ui.(string)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Retrieve the user from the database
	user, exists, err := handler.Driver.GetUser(username)
	if !exists || err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	contents, err := json.Marshal(user)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(contents)
}

// Inserts the provided user into the database
// This acts as the signup endpoint -- TODO: Change name accordingly
// PUT /v1/users
func (handler *CoreHandler) InsertUser(w http.ResponseWriter, r *http.Request) {
	// TODO: dont think the below line is needed any more
	w.Header().Set("Access-Control-Allow-Origin", "*")
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}

	log.Printf("Received the following user to signup: %v", string(body))

	// Marshal the body into a user object
	user := &models.User{}
	if err := json.Unmarshal(body, user); err != nil {
		log.Printf("Error parsing body: %v", err)
		http.Error(w, "can't parse body", http.StatusBadRequest)
		return
	}

	// TODO Verify username is unique

	// We must insert a custom generate UUID into the user
	user.ID = uuid.NewV4().String()

	// Hash our password
	user.Password, err = HashPassword(user.Password)
	if err != nil {
		http.Error(w, "error inserting user", http.StatusInternalServerError)
		return
	}

	handler.Driver.InsertUser(*user)
	w.WriteHeader(http.StatusOK)
}

func writePosts(w http.ResponseWriter, posts []models.Post) {
	w.Header().Set("Content-Type", "application/json")
	// Write our posts as a response
	res, err := json.Marshal(posts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(res)
}

// gets a list of content providers, one for each supported client. A content
// provider structure stores information about the current page of data being
// read from that content provider, and a function to get the next page of data.
func (handler *CoreHandler) getContentProviders(user models.User) []*ranking.ContentProvider {
	providers := []*ranking.ContentProvider{}
	// Construct a buffered channel to hold posts from each of our clients
	ch := make(chan *ranking.ContentProvider, len(handler.Clients))
	for _, c := range handler.Clients {
		generator, err := c.GetPageGenerator(user)
		if err != nil {
			log.Printf("error getting page generator for %v: %v", c.Name(), err)
			continue
		}

		go func(ch chan *ranking.ContentProvider) {
			ch <- ranking.NewContentProvider(c.Weight(), generator)
		}(ch)
	}

	// Note: The number of arguments to append has to be kept up to date with the len(handler.Clients) for everything to work
	providers = append(providers, <-ch, <-ch, <-ch, <-ch)

	return providers
}

// GET /v1/posts
func (handler *CoreHandler) GetPosts(w http.ResponseWriter, r *http.Request) {
	s, err := handler.SessionManager.GetSession(r)
	if err != nil {
		// Return unauthorized error -- but TODO: in the future differentiate between 401 and 500
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Get user associate with the session
	username := s.Get("username").(string)

	// Retrieve the user from the database
	user, exists, err := handler.Driver.GetUser(username)
	if !exists || err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var providers []*ranking.ContentProvider
	query := r.URL.Query()
	if token, ok := query["page_token"]; ok && len(token) != 0 {
		if p, ok := handler.Cache.Get(token[0]); ok {
			providers = p.([]*ranking.ContentProvider)
		} else {
			http.Error(w, "Data for page token not found", http.StatusNotFound)
			return
		}
	} else {
		providers = handler.getContentProviders(user)
	}

	posts := ranking.GetPosts(providers, 20)
	pageToken := handler.getNextPagingToken()
	handler.Cache.Set(pageToken, providers, cache.DefaultExpiration)

	w.Header().Set("Content-Type", "application/json")
	res, err := json.Marshal(
		struct {
			Posts     []models.Post `json:"posts"`
			PageToken string        `json:"page_token"`
		}{posts, pageToken})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(res)
}
