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

func New(Host string, Port int, Weight float64) *Reddit {
	return &Reddit{Host, Port, Weight}
}

func (r *Reddit) GetPageGenerator(user models.User) (func() []models.Post, error) {
	// TODO: Eventually we will first have to check whether this token exists or if it expired
	if user.RedditAuthToken == "" {
		return nil, fmt.Errorf("Unable to retrieve reddit oauth token from database for user: %v\n", user.RedditUsername)
	}

	nextURL := fmt.Sprintf("http://%v:%v/v1/%v/posts", r.Host, r.Port, user.RedditUsername)
	getNextPage := func() []models.Post {
		if nextURL == "" {
			return []models.Post{}
		}
		resp := r.getPosts(nextURL, user.RedditAuthToken)
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
	return "Reddit"
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
	log.Printf("resp: \n%+v\n", clientResp)
	posts = clientResp.Posts
	if err != nil {
		return clients.PostResponse{posts, "", fmt.Errorf("Unable to decode posts from Reddit: %v", err)}
	}

	log.Println("Successfully retrieved posts from reddit")
	return clients.PostResponse{posts, clientResp.NextURL, nil}
}
