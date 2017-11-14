package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"
	"github.com/iced-mocha/core/config"
	"github.com/iced-mocha/core/ranking"
	"github.com/iced-mocha/core/sessions"
	"github.com/iced-mocha/core/storage"
	"github.com/iced-mocha/shared/models"
	"github.com/satori/go.uuid"
	"golang.org/x/crypto/bcrypt"
)

type CoreHandler struct {
	Driver         storage.Driver
	Config         config.Config
	SessionManager sessions.Manager

	redditHost, facebookHost, hnHost, gnHost string
	redditPort, facebookPort, hnPort, gnPort int
}

// Structure received when updating reddit oauth token
type RedditAuth struct {
	User         string
	BearerToken  string
	RefreshToken string
}

// Wrapper for the response from a post client
type PostResponse struct {
	posts []models.Post
	nextURL  string
	err   error
}

func New(d storage.Driver, sm sessions.Manager, c config.Config) (*CoreHandler, error) {
	handler := &CoreHandler{}
	handler.Driver = d
	handler.Config = c
	handler.SessionManager = sm

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

	handler.hnHost, handler.facebookHost, handler.redditHost, handler.gnHost = hosts[0], hosts[1], hosts[2], hosts[3]
	handler.hnPort, handler.facebookPort, handler.redditPort, handler.gnPort = ports[0], ports[1], ports[2], ports[3]

	return handler, nil
}

// Updates the reddit oauth token stored for a user with id <userID>
// POST /v1/user/{userID}/authorize/reddit
func (handler *CoreHandler) UpdateRedditAuth(w http.ResponseWriter, r *http.Request) {
	// Read body of the request
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}

	log.Printf("Body: %v\n", string(body))

	// Get the user id from path paramater
	id := mux.Vars(r)["userID"]

	// Change the body into a user object
	auth := &RedditAuth{}
	err = json.Unmarshal(body, auth)
	if err != nil {
		log.Printf("Error parsing body: %v", err)
		http.Error(w, "can't parse body", http.StatusBadRequest)
		return
	}

	handler.Driver.UpdateOAuthToken(id, auth.BearerToken, "")
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

func (handler *CoreHandler) Login(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		http.Error(w, "unable to read request body", http.StatusBadRequest)
		return
	}

	// NOTE: We expect a user object but username and password are all that will be non-empty
	log.Printf("Received the following user to login: %v", string(body))

	// Marshal the body into a user object
	attemptedUser := &models.User{}
	if err := json.Unmarshal(body, attemptedUser); err != nil {
		log.Printf("Error parsing body: %v", err)
		http.Error(w, "can't parse body", http.StatusBadRequest)
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
		http.Error(w, "unable ot get user", http.StatusInternalServerError)
		return
	}

	// If the user does not exist return 401 (Unauthorized) for security reasons
	if !exists {
		log.Printf("Requested user %v does not exist", attemptedUser.Username)
		http.Error(w, "bad credentials", http.StatusUnauthorized)
		return
	}

	// Otherwise the user exists so lets see if we were provided correct credentials
	// NOTE: attemptedUser.Password is plaintext and actualUser.Password is bcrypted hash of password
	valid := CheckPasswordHash(attemptedUser.Password, actualUser.Password)
	if !valid {
		// Not valid so return unauthorized
		log.Printf("Bad credentials attempting to authenticate user %v", attemptedUser.Username)
		http.Error(w, "bad credentials", http.StatusUnauthorized)
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

// Fetches posts from hackernews
func (handler *CoreHandler) getHackerNewsPosts(c chan PostResponse) {
	hnPosts := make([]models.Post, 0)

	hnResp, err := http.Get(fmt.Sprintf("http://%v:%v/v1/posts?count=20", handler.hnHost, handler.hnPort))
	if err != nil {
		c <- PostResponse{hnPosts, "", fmt.Errorf("Unable to fetch posts from hacker news: %v", err)}
		return
	}
	defer hnResp.Body.Close()

	var hnRespBody models.ClientResp
	err = json.NewDecoder(hnResp.Body).Decode(&hnRespBody)
	if err != nil {
		c <- PostResponse{hnPosts, "", fmt.Errorf("Unable to decode response from hacker-news: %v", err)}
		return
	}

	log.Println("Successfully retrieved posts from hackernews")
	c <- PostResponse{hnRespBody.Posts, hnRespBody.NextURL, nil}
}

func (handler *CoreHandler) getFacebookPosts(query url.Values, c chan PostResponse) {
	var fbId string
	var fbToken string

	// TODO what if these are empty?
	if v, ok := query["fb_id"]; ok && len(v) != 0 {
		fbId = v[0]
	}
	if v, ok := query["fb_token"]; ok && len(v) != 0 {
		fbToken = v[0]
	}

	var fbRespBody models.ClientResp
	var fbPosts = make([]models.Post, 0)
	fbResp, err := http.Get(fmt.Sprintf("http://%v:%v/v1/posts?fb_id=%v&fb_token=%v", handler.facebookHost, handler.facebookPort, fbId, fbToken))
	if err != nil {
		c <- PostResponse{fbPosts, "", fmt.Errorf("Unable to get posts from facebook: %v", err)}
		return
	}
	defer fbResp.Body.Close()

	err = json.NewDecoder(fbResp.Body).Decode(&fbRespBody)
	fbPosts = fbRespBody.Posts
	if err != nil {
		c <- PostResponse{fbPosts, "", fmt.Errorf("Unable to decode posts from facebook: %v", err)}
		return
	}

	log.Println("Successfully retrieved posts from facebook")
	c <- PostResponse{fbPosts, fbRespBody.NextURL, nil}
}

func (handler *CoreHandler) getRedditPosts(c chan PostResponse) {
	// Use this userID until we implement login
	username := "userID"
	redditPosts := make([]models.Post, 0)

	// TODO: Eventually we will first have to check whether this token exists or if it expired
	// Get our reddit bearer token
	redditToken, err := handler.Driver.GetRedditOAuthToken(username)
	if err != nil || redditToken == "" {
		c <- PostResponse{redditPosts, "", fmt.Errorf("Unable to retrieve reddit oauth token from database for user: %v\n Error:  %v", username, err)}
		return
	}

	client := &http.Client{}
	log.Printf("Token:%v\n", redditToken)
	jsonString := []byte(fmt.Sprintf("{ \"bearertoken\": \"%v\"}", redditToken))
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%v:%v/v1/%v/posts",
		handler.redditHost, handler.redditPort, username), bytes.NewBuffer(jsonString))
	if err != nil {
		c <- PostResponse{redditPosts, "", err}
		return
	}

	redditResp, err := client.Do(req)
	if err != nil {
		c <- PostResponse{redditPosts, "", fmt.Errorf("Unable to get posts from reddit: %v", err)}
		return
	}
	defer redditResp.Body.Close()

	err = json.NewDecoder(redditResp.Body).Decode(&redditPosts)
	if err != nil {
		c <- PostResponse{redditPosts, "", fmt.Errorf("Unable to decode posts from Reddit: %v", err)}
		return
	}

	log.Println("Successfully retrieved posts from reddit")
	c <- PostResponse{redditPosts, "", nil} // TODO: Don't return "", return the pagination URL
}

func (handler *CoreHandler) getGoogleNewsPosts(c chan PostResponse) {
	gnPosts := make([]models.Post, 0, 0)

	gnResp, err := http.Get(fmt.Sprintf("http://%v:%v/v1/posts?count=20", handler.gnHost, handler.gnPort))
	if err != nil {
		c <- PostResponse{gnPosts, "", fmt.Errorf("Unable to fetch posts from google news: %v", err)}
		return
	}
	defer gnResp.Body.Close()

	err = json.NewDecoder(gnResp.Body).Decode(&gnPosts)
	if err != nil {
		c <- PostResponse{gnPosts, "", fmt.Errorf("Unable to decode response from google-news: %v", err)}
		return
	}

	log.Println("Successfully retrieved posts from googlenews")
	c <- PostResponse{gnPosts, "", nil}
}

func combinePosts(postResponses []PostResponse) []models.Post {
	posts := make([]models.Post, 0)
	for _, val := range postResponses {
		if val.err != nil {
			log.Printf("Unable to get posts: %v", val.err)
		} else {
			posts = append(posts, val.posts...)
		}
	}
	return posts
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

/*
func (handler *CoreHandler) GetNoAuthPosts(w http.ResponseWriter, r *http.Request) {
	c := make(chan PostResponse, 2)

	go handler.getHackerNewsPosts(c)
	go handler.getGoogleNewsPosts(c)

	r1, r2 := <-c, <-c
	postResponses := []PostResponse{r1, r2}
	posts := combinePosts(postResponses)

	// TODO Can this be made a constant outside the function?
	weights := make(map[string]float64)
	weights[models.PlatformHackerNews] = 4
	weights[models.PlatformGoogleNews] = 8
	sort.Sort(comparators.ByPostRank{posts, weights})

	writePosts(w, posts)
}
*/

// GET /v1/posts
func (handler *CoreHandler) GetPosts(w http.ResponseWriter, r *http.Request) {
	// TODO: different logic is needing depending if we are logged in or not
	//if handler.SessionManager.HasSession(r) {
	getNextHNPage := func() []models.Post {
		// TODO: should get next page, not same page over and over
		c := make(chan PostResponse)
		go handler.getHackerNewsPosts(c)
		resp := <-c
		if resp.err == nil {
			return resp.posts
		} else {
			log.Printf("error getting hn page %v\n", resp.err)
			return make([]models.Post, 0)
		}
	}

	getNextFBPage := func() []models.Post {
		// TODO: should get next page, not same page over and over
		c := make(chan PostResponse)
		go handler.getFacebookPosts(r.URL.Query(), c)
		resp := <-c
		if resp.err == nil {
			return resp.posts
		} else {
			log.Printf("error getting fb page %v\n", resp.err)
			return make([]models.Post, 0)
		}
	}

	getNextRDPage := func() []models.Post {
		// TODO: should get next page, not same page over and over
		c := make(chan PostResponse)
		go handler.getRedditPosts(c)
		resp := <-c
		if resp.err == nil {
			return resp.posts
		} else {
			log.Printf("error getting rd page %v\n", resp.err)
			return make([]models.Post, 0)
		}
	}

	getNextGNPage := func() []models.Post {
		// TODO: should get next page, not same page over and over
		c := make(chan PostResponse)
		go handler.getGoogleNewsPosts(c)
		resp := <-c
		if resp.err == nil {
			return resp.posts
		} else {
			log.Printf("error getting gn page %v\n", resp.err)
			return make([]models.Post, 0)
		}
	}

	hn := ranking.NewContentProvider(4, getNextHNPage)
	fb := ranking.NewContentProvider(1, getNextFBPage)
	rd := ranking.NewContentProvider(4, getNextRDPage)
	gn := ranking.NewContentProvider(8, getNextGNPage)

	posts := ranking.GetPosts([]*ranking.ContentProvider{hn, fb, rd, gn}, 40)

	writePosts(w, posts)
}
