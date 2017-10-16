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
	"github.com/iced-mocha/core/storage/driver"
	"github.com/iced-mocha/core/storage/driver/sqlite"
	"github.com/iced-mocha/shared/models"
)

type CoreHandler struct{}

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

var d driver.StorageDriver

func init() {
	var err error

	d, err = sqlite.New(sqlite.Config{})
	if err != nil {
		log.Printf("Unable to create driver: %v\n", err)
	}
}

// Function for updating auth token in datastore
// /v1/user/{userID}/authorize/reddit POST
func (api *CoreHandler) UpdateRedditAuth(w http.ResponseWriter, r *http.Request) {
	// Read body of the request
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}

	// Get the user id from path paramater
	vars := mux.Vars(r)

	// Change the body into a user object
	auth := &RedditAuth{}
	log.Printf("Body: %v\n", string(body))
	err = json.Unmarshal(body, auth)
	if err != nil {
		log.Printf("Error parsing body: %v", err)
		http.Error(w, "can't parse body", http.StatusBadRequest)
		return
	}

	d.UpdateOAuthToken(vars["userID"], auth.BearerToken, "")
	w.WriteHeader(http.StatusOK)
}

func (api *CoreHandler) RedditAuth(w http.ResponseWriter, r *http.Request) {
	// Rediret to reddit-client auth
	http.Redirect(w, r, "http://localhost:3001/v1/authorize", http.StatusFound)
}

func (api *CoreHandler) InsertUser(w http.ResponseWriter, r *http.Request) {
	// Read body of the request
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}

	// Change the body into a user object
	user := &models.User{}
	err = json.Unmarshal(body, user)
	if err != nil {
		log.Printf("Error parsing body: %v", err)
		http.Error(w, "can't parse body", http.StatusBadRequest)
		return
	}

	// TODO Add userID to be path param
	userId := "userID"
	d.InsertUser(userId, user.Name)
	log.Println("userId: " + userId)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(userId))

}

// Fetches posts from hackernews
func (api *CoreHandler) getHackerNewsPosts(c chan PostResponse) {
	// Create an inital array with the same amount of posts we expect to get from hackernews
	hnPosts := make([]models.Post, 20)

	hnResp, err := http.Get("http://hacker-news-client:4000/v1/posts?count=20")
	if err != nil {
		c <- PostResponse{hnPosts, fmt.Errorf("Unable to fetch posts from hacker news: %v", err)}
		return
	}

	err = json.NewDecoder(hnResp.Body).Decode(&hnPosts)
	if err != nil {
		c <- PostResponse{hnPosts, fmt.Errorf("Unable to decode response from hacker-news: %v", err)}
		return
	}

	c <- PostResponse{hnPosts, nil}
}

func (api *CoreHandler) getFacebookPosts(query url.Values, c chan PostResponse) {
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
	fbResp, err := http.Get("http://facebook-client:5000/v1/posts?fb_id=" + fbId + "&fb_token=" + fbToken)
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

	c <- PostResponse{fbPosts, nil}
}

func (api *CoreHandler) getRedditPosts(c chan PostResponse) {
	// Use this userID until we implement login
	userID := "userID"
	redditPosts := make([]models.Post, 0)

	// TODO: Eventually we will first have to check whether this token exists or if it expired
	// Get our reddit bearer token
	redditToken, err := d.GetRedditOAuthToken(userID)
	if err != nil || redditToken == "" {
		c <- PostResponse{redditPosts, fmt.Errorf("Unable to oauth token from database for user: %v: %v", userID, err)}
		return
	}

	client := &http.Client{}
	log.Printf("Token:%v\n", redditToken)
	jsonString := []byte(fmt.Sprintf("{ \"bearertoken\": \"%v\"}", redditToken))
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:3001/v1/%v/posts", userID), bytes.NewBuffer(jsonString))
	if err != nil {
		c <- PostResponse{redditPosts, err}
		return
	}

	redditResp, err := client.Do(req)
	if err != nil {
		c <- PostResponse{redditPosts, fmt.Errorf("Unable to get posts form reddit: %v", err)}
		return
	}

	err = json.NewDecoder(redditResp.Body).Decode(&redditPosts)
	if err != nil {
		c <- PostResponse{redditPosts, fmt.Errorf("Unable to decode posts from Reddit: %v", err)}
		return
	}

	c <- PostResponse{redditPosts, nil}
}

// GET /v1/posts
func (api *CoreHandler) GetPosts(w http.ResponseWriter, r *http.Request) {

	// Make a channel for the various posts we are getting
	// Right now we have 3 clients to fetch from so we need to block until all 3 complete
	c := make(chan PostResponse, 3)

	// Get the response from hackernews
	go api.getHackerNewsPosts(c)
	go api.getFacebookPosts(r.URL.Query(), c)
	go api.getRedditPosts(c)

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
