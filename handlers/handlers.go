package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sort"

	"github.com/gorilla/mux"
	"github.com/iced-mocha/core/comparators"
	"github.com/iced-mocha/core/config"
	"github.com/iced-mocha/core/storage/driver"
	"github.com/iced-mocha/shared/models"
	"github.com/satori/go.uuid"
)

type CoreHandler struct {
	Driver driver.StorageDriver
	Config config.Config

	redditHost, facebookHost, hnHost, gnHost string
	redditPort, facebookPort, hnPort, gnPort int
}

// Structur received when updating reddit oauth token
type RedditAuth struct {
	User         string
	BearerToken  string
	RefreshToken string
}

// Wrapper for the response from a post client
type PostResponse struct {
	posts []models.Post
	err   error
}

func New(d driver.StorageDriver, c config.Config) (*CoreHandler, error) {
	handler := &CoreHandler{}
	handler.Driver = d
	handler.Config = c

	// TODO Find a better way to do this
	// Maybe create a GetStringKeys function that returns array of values and a potential error

	hosts, err := handler.Config.GetStrings([]string{"hacker-news.host", "facebook.host", "reddit.host"})
	if err != nil {
		return nil, err
	}

	ports, err := handler.Config.GetInts([]string{"hacker-news.port", "facebook.port", "reddit.port"})
	if err != nil {
		return nil, err
	}

	handler.hnHost, handler.facebookHost, handler.redditHost = hosts[0], hosts[1], hosts[2]
	handler.hnPort, handler.facebookPort, handler.redditPort = ports[0], ports[1], ports[2]

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

// Inserts the provided user into the database
// PUT /v1/users
func (handler *CoreHandler) InsertUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}

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

	//userId := "userID"
	handler.Driver.InsertUser(*user)
	log.Println("userId: " + user.ID)
	w.WriteHeader(http.StatusOK)
}

// Fetches posts from hackernews
func (handler *CoreHandler) getHackerNewsPosts(c chan PostResponse) {
	// Create an inital array with the same amount of posts we expect to get from hackernews
	hnPosts := make([]models.Post, 20)

	hnResp, err := http.Get(fmt.Sprintf("http://%v:%v/v1/posts?count=20", handler.hnHost, handler.hnPort))
	if err != nil {
		c <- PostResponse{hnPosts, fmt.Errorf("Unable to fetch posts from hacker news: %v", err)}
		return
	}

	err = json.NewDecoder(hnResp.Body).Decode(&hnPosts)
	if err != nil {
		c <- PostResponse{hnPosts, fmt.Errorf("Unable to decode response from hacker-news: %v", err)}
		return
	}

	log.Println("Successfully retrieved posts from hackernews")
	c <- PostResponse{hnPosts, nil}
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
		c <- PostResponse{fbPosts, fmt.Errorf("Unable to get posts from facebook: %v", err)}
		return
	}

	err = json.NewDecoder(fbResp.Body).Decode(&fbRespBody)
	fbPosts = fbRespBody.Posts
	if err != nil {
		c <- PostResponse{fbPosts, fmt.Errorf("Unable to decode posts from facebook: %v", err)}
		return
	}

	log.Println("Successfully retrieved posts from facebook")
	c <- PostResponse{fbPosts, nil}
}

func (handler *CoreHandler) getRedditPosts(c chan PostResponse) {
	// Use this userID until we implement login
	username := "userID"
	redditPosts := make([]models.Post, 0)

	// TODO: Eventually we will first have to check whether this token exists or if it expired
	// Get our reddit bearer token
	redditToken, err := handler.Driver.GetRedditOAuthToken(username)
	if err != nil || redditToken == "" {
		c <- PostResponse{redditPosts, fmt.Errorf("Unable to retrieve oauth token from database for user: %v\n Error:  %v", username, err)}
		return
	}

	client := &http.Client{}
	log.Printf("Token:%v\n", redditToken)
	jsonString := []byte(fmt.Sprintf("{ \"bearertoken\": \"%v\"}", redditToken))
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%v:%v/v1/%v/posts", handler.redditHost, handler.redditPort, username), bytes.NewBuffer(jsonString))
	if err != nil {
		c <- PostResponse{redditPosts, err}
		return
	}

	redditResp, err := client.Do(req)
	if err != nil {
		c <- PostResponse{redditPosts, fmt.Errorf("Unable to get posts from reddit: %v", err)}
		return
	}

	err = json.NewDecoder(redditResp.Body).Decode(&redditPosts)
	if err != nil {
		c <- PostResponse{redditPosts, fmt.Errorf("Unable to decode posts from Reddit: %v", err)}
		return
	}

	log.Println("Successfully retrieved posts from reddit")
	c <- PostResponse{redditPosts, nil}
}

// GET /v1/posts
func (handler *CoreHandler) GetPosts(w http.ResponseWriter, r *http.Request) {

	// Make a channel for the various posts we are getting
	// Right now we have 3 clients to fetch from so we need to block until all 3 complete
	c := make(chan PostResponse, 3)

	// Get the response from hackernews
	go handler.getHackerNewsPosts(c)
	go handler.getFacebookPosts(r.URL.Query(), c)
	go handler.getRedditPosts(c)

	// Read all response from channel (we dont really know which is which)
	r1, r2, r3 := <-c, <-c, <-c

	// Error check based on the responses from each of of request
	if r1.err != nil && r2.err != nil && r3.err != nil {
		http.Error(w, "Could not receive posts from any of our client", http.StatusInternalServerError)
		log.Printf("Errors:\n %v \n %v \n %v", r1.err, r2.err, r3.err)
		return
	}

	posts := make([]models.Post, 0)

	// Otherwise if we reach here at least one of our clients was successful so lets take the posts we did get
	// TODO: Make this better
	if r1.err == nil {
		posts = append(posts, r1.posts...)
	}
	if r2.err == nil {
		posts = append(posts, r2.posts...)
	}
	if r3.err == nil {
		posts = append(posts, r3.posts...)
	}

	weights := make(map[string]float64)
	weights[models.PlatformHackerNews] = 4
	weights[models.PlatformReddit] = 4
	weights[models.PlatformFacebook] = 1
	sort.Sort(comparators.ByPostRank{posts, weights})

	w.Header().Set("Content-Type", "application/json")

	// Write our posts as a response
	res, err := json.Marshal(posts)
	if err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write(res)
	} else {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
