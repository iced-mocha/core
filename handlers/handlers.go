package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
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
	"github.com/iced-mocha/core/clients/rss"
	"github.com/iced-mocha/core/clients/twitter"
	"github.com/iced-mocha/core/config"
	"github.com/iced-mocha/core/creds"
	"github.com/iced-mocha/core/ranking"
	"github.com/iced-mocha/core/sessions"
	"github.com/iced-mocha/core/storage"
	"github.com/iced-mocha/shared/models"
	"github.com/patrickmn/go-cache"
	"github.com/satori/go.uuid"
)

const (
	DefaultRedditWeight     = 20.0
	DefaultTwitterWeight    = 20.0
	DefaultFacebookWeight   = 10.0
	DefaultHackerNewsWeight = 20.0
	DefaultGoogleNewsWeight = 20.0
)

var DefaultRssWeights = map[string]float64{
	"news":   13.0,
	"sports": 10.0,
}

var DefaultRssGroups = map[string][]string{
	"news": []string{
		"http://feeds.bbci.co.uk/news/rss.xml",
		"http://www.cbc.ca/cmlink/rss-topstories",
	},
	"sports": []string{
		"http://www.espn.com/espn/rss/news",
		"https://api.foxsports.com/v1/rss",
	},
}

type CoreHandler struct {
	Driver         storage.Driver
	Config         config.Config
	SessionManager sessions.Manager
	// TODO: Having a cache on core used for pagination requires us to only run
	// one instance of core
	Cache              *cache.Cache
	getNextPagingToken func() string

	Clients   []clients.Client
	RssClient *rss.RSS
}

// Structure received when updating provider auth info
type ProviderAuth struct {
	Type         string `json:"type"`
	Username     string `json:"username"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh-token"`
	Secret       string `json:"secret"`
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

	hosts, err := handler.Config.GetStrings([]string{"hacker-news.host", "facebook.host", "reddit.host", "google-news.host", "twitter.host", "rss.host"})
	if err != nil {
		return nil, err
	}

	ports, err := handler.Config.GetInts([]string{"hacker-news.port", "facebook.port", "reddit.port", "google-news.port", "twitter.port", "rss.port"})
	if err != nil {
		return nil, err
	}

	handler.Clients = make([]clients.Client, 5)
	handler.Clients[0] = hackernews.New(hosts[0], ports[0])
	handler.Clients[1] = facebook.New(hosts[1], ports[1])
	handler.Clients[2] = reddit.New(hosts[2], ports[2])
	handler.Clients[3] = googlenews.New(hosts[3], ports[3])
	handler.Clients[4] = twitter.New(hosts[4], ports[4])
	handler.RssClient = rss.New(hosts[5], ports[5])

	return handler, nil
}

// POST /v1/weights
func (h *CoreHandler) UpdateWeights(w http.ResponseWriter, r *http.Request) {
	log.Printf("received request at POST /v1/weights")
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
	log.Printf("Preparing to update weights for user: %v", username)

	// Now get the user for the username
	u, exists, err := h.Driver.GetUser(username)
	if !exists || err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Unmarshal our response body so we can access the given weights
	contents, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	weights := &models.Weights{}
	err = json.Unmarshal(contents, weights)
	if err != nil {
		log.Printf("Unable to marshal weights object: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Printf("Received the follow weights for user %v: %+v\n", u.Username, weights)

	// Insert our user into the db
	// This violates unique constraint
	b := h.Driver.UpdateWeights(u.Username, *weights)
	if !b {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Deletes the type of linked account for authenticated user in the request
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
	} else if t == "twitter" {
		h.Driver.UpdateTwitterAccount(username, "", "", "")
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

// Updates the twitter oauth token and secret stored for a user with id <userID>
// POST /v1/user/{userID}/authorize/twitter
func (handler *CoreHandler) UpdateTwitterAuth(w http.ResponseWriter, r *http.Request) {
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

	log.Printf("received the following twitter auth: %+v", auth)

	successful := handler.Driver.UpdateTwitterAccount(id, auth.Username, auth.Token, auth.Secret)
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
	userID := mux.Vars(r)["userID"]
	// TODO rip out this config
	host, _ := handler.Config.GetString("reddit.host")
	port, _ := handler.Config.GetInt("reddit.port")
	http.Redirect(w, r, "https://"+host+":"+strconv.Itoa(port)+"/v1/"+userID+"/authorize", http.StatusMovedPermanently)
}

// Redirects to our twitter client to authorize our service to use the users twitter account
// GET /v1/user/{userID}/authorize/twitter
func (handler *CoreHandler) TwitterAuth(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["userID"]
	// TODO rip out this config
	host, _ := handler.Config.GetString("twitter.host")
	port, _ := handler.Config.GetInt("twitter.port")
	http.Redirect(w, r, "https://"+host+":"+strconv.Itoa(port)+"/v1/"+userID+"/authorize", http.StatusMovedPermanently)
}

func (handler *CoreHandler) IsLoggedIn(w http.ResponseWriter, r *http.Request) {
	// First check to see if the user is already logged in
	_, err := handler.SessionManager.GetSession(r)
	if err != nil {
		// Destroy the invalid session
		handler.SessionManager.SessionDestroy(w, r)
		w.Write([]byte(`{ "logged-in": false }`))
		return
	}
	w.Write([]byte(`{ "logged-in": true }`))
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
	valid := creds.CheckPasswordHash(attemptedUser.Password, actualUser.Password)
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

	// verify username and password meet out criteria of valid
	if err := creds.ValidateSignupCredentials(user.Username, user.Password); err != nil {
		log.Printf("Attempted to sign up user %v with invalid credentials - %v", user.Username, err)
		http.Error(w, buildJSONError(err.Error()), http.StatusBadRequest)
		return
	}

	_, exists, err := handler.Driver.GetUser(user.Username)
	if exists || err != nil {
		log.Printf("Attempted to sign up user %v but username already exists", user.Username)
		http.Error(w, buildJSONError(fmt.Sprintf("Attempted to sign up with username: %v - but username already exists", user.Username)),
			http.StatusBadRequest)
		return
	}

	// We must insert a custom generate UUID into the user
	user.ID = uuid.NewV4().String()

	// Hash our password
	user.Password, err = creds.HashPassword(user.Password)
	if err != nil {
		http.Error(w, "error inserting user", http.StatusInternalServerError)
		return
	}

	// Add the default weighting to the user struct
	pw := models.Weights{}
	pw.Reddit = DefaultRedditWeight
	pw.Facebook = DefaultFacebookWeight
	pw.HackerNews = DefaultHackerNewsWeight
	pw.GoogleNews = DefaultGoogleNewsWeight
	pw.Twitter = DefaultTwitterWeight
	pw.RSS = DefaultRssWeights
	user.PostWeights = pw
	user.RssGroups = DefaultRssGroups

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

func getDefaultWeight(clientName string) float64 {
	var val int

	if clientName == "reddit" {
		val = DefaultRedditWeight
	} else if clientName == "facebook" {
		val = DefaultFacebookWeight
	} else if clientName == "hacker-news" {
		val = DefaultHackerNewsWeight
	} else if clientName == "google-news" {
		val = DefaultGoogleNewsWeight
	} else if clientName == "twitter" {
		val = DefaultTwitterWeight
	}

	return float64(val)
}

func getWeight(clientName string, user *models.User) float64 {
	if user == nil {
		return getDefaultWeight(clientName)
	}

	var val float64

	if clientName == "reddit" {
		val = user.PostWeights.Reddit
	} else if clientName == "facebook" {
		val = user.PostWeights.Facebook
	} else if clientName == "hacker-news" {
		val = user.PostWeights.HackerNews
	} else if clientName == "google-news" {
		val = user.PostWeights.GoogleNews
	} else if clientName == "twitter" {
		val = user.PostWeights.Twitter
	}

	return float64(val)
}

// gets a list of content providers, one for each supported client. A content
// provider structure stores information about the current page of data being
// read from that content provider, and a function to get the next page of data.
func (handler *CoreHandler) getContentProviders(user *models.User) []*ranking.ContentProvider {
	providers := []*ranking.ContentProvider{}
	// Construct a buffered channel to hold posts from each of our clients
	numContentProviders := 0
	ch := make(chan *ranking.ContentProvider)
	for _, c := range handler.Clients {
		generator, err := c.GetPageGenerator(user)
		if err != nil {
			log.Printf("error getting page generator for %v: %v", c.Name(), err)
			continue
		}

		numContentProviders++
		go func(name string, ch chan *ranking.ContentProvider) {
			// Sometimes user is nil when we are not authenticated
			tw := getWeight(name, user)
			ch <- ranking.NewContentProvider(tw, generator)
		}(c.Name(), ch)
	}

	rssGroups := DefaultRssGroups
	rssWeights := DefaultRssWeights
	if user != nil {
		rssGroups = user.RssGroups
		rssWeights = user.PostWeights.RSS
	}

	for name, group := range rssGroups {
		generator, err := handler.RssClient.GetPageGenerator(group)
		if err != nil {
			log.Printf("error getting page generator for rss group %v: %v", name, err)
			continue
		}

		numContentProviders++
		go func(name string, ch chan *ranking.ContentProvider) {
			ch <- ranking.NewContentProvider(rssWeights[name], generator)
		}(name, ch)
	}

	for i := 0; i < numContentProviders; i++ {
		providers = append(providers, <-ch)
	}

	return providers
}

// GET /v1/posts
func (handler *CoreHandler) GetPosts(w http.ResponseWriter, r *http.Request) {
	s, err := handler.SessionManager.GetSession(r)
	if err != nil {
		handler.getPostsForUser(nil, w, r)
		return
	}

	// Get user associate with the session
	username := s.Get("username").(string)

	// Retrieve the user from the database
	user, _, err := handler.Driver.GetUser(username)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	handler.getPostsForUser(&user, w, r)
}

func (handler *CoreHandler) GetProviders(user *models.User, r *http.Request) ([]*ranking.ContentProvider, error) {
	token := r.FormValue("page_token")
	if token != "" {
		if p, ok := handler.Cache.Get(token); ok {
			providers, ok := p.([]*ranking.ContentProvider)
			if !ok {
				log.Printf("Data associated to page token: %v malformed", token)
				return nil, errors.New("malformed paging data")
			}

			// Able to retrieve providers from cache
			return providers, nil
		}

		log.Printf("Unable to find data associated to page token: %v", token)
		return nil, errors.New("data associated with page token not found")
	}

	return handler.getContentProviders(user), nil
}

func (handler *CoreHandler) getPostsForUser(user *models.User, w http.ResponseWriter, r *http.Request) {
	providers, err := handler.GetProviders(user, r)
	if err != nil {
		log.Printf("Unable to get content providers")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	posts := ranking.GetPosts(providers, 20)
	pageToken := handler.getNextPagingToken()
	handler.Cache.Set(pageToken, providers, cache.DefaultExpiration)
	log.Printf("Received %v posts from content providers", len(posts))

	w.Header().Set("Content-Type", "application/json")
	res, err := json.Marshal(
		// TODO define this outside this function
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
