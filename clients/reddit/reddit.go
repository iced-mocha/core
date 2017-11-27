package reddit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/iced-mocha/core/clients"
	"github.com/iced-mocha/shared/models"
)

type Reddit struct {
	Host   string
	Port   int
	weight float64
}

func New(host string, port int) *Reddit {
	return &Reddit{Host: host, Port: port}
}

func (r *Reddit) GetPageGenerator(user *models.User) (func() []models.Post, error) {
	var authToken string
	var nextURL string
	if user == nil || user.RedditUsername == "" {
		authToken = ""
		nextURL = fmt.Sprintf("http://%v:%v/v1/posts", r.Host, r.Port)
	} else {
		authToken = user.RedditAuthToken
		// TODO: Eventually we will first have to check if the reddit token will expire
		nextURL = fmt.Sprintf("http://%v:%v/v1/%v/posts", r.Host, r.Port, user.RedditUsername)
	}
	getNextPage := func() []models.Post {
		if nextURL == "" {
			return []models.Post{}
		}
		resp := r.getPosts(nextURL, authToken)
		if resp.Err == nil {
			nextURL = resp.NextURL
			return resp.Posts
		} else {
			nextURL = ""
			log.Printf("error getting rd page %v\n", resp.Err)
			return []models.Post{}
		}
	}

	return getNextPage, nil
}

func (r *Reddit) Name() string {
	return "reddit"
}

func (r *Reddit) Weight() float64 {
	return r.weight
}

func (r *Reddit) getPosts(url, redditToken string) clients.PostResponse {
	posts := []models.Post{}
	client := &http.Client{}
	log.Printf("Token:%v\n", redditToken)
	jsonString := []byte(fmt.Sprintf("{ \"bearertoken\": \"%v\"}", redditToken))
	req, err := http.NewRequest(http.MethodGet, url, bytes.NewBuffer(jsonString))
	if err != nil {
		return clients.PostResponse{posts, "", err}
	}

	redditResp, err := client.Do(req)
	if err != nil {
		return clients.PostResponse{posts, "", fmt.Errorf("Unable to get posts from reddit: %v", err)}
	}
	defer redditResp.Body.Close()

	if redditResp.StatusCode != http.StatusOK {
		return clients.PostResponse{posts, "", fmt.Errorf("Unable to get posts from reddit received status code: %v", redditResp.StatusCode)}
	}

	clientResp := models.ClientResp{}
	err = json.NewDecoder(redditResp.Body).Decode(&clientResp)
	posts = clientResp.Posts
	if err != nil {
		return clients.PostResponse{posts, "", fmt.Errorf("Unable to decode posts from Reddit: %v", err)}
	}

	log.Println("Successfully retrieved posts from reddit")
	return clients.PostResponse{posts, clientResp.NextURL, nil}
}
