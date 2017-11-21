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
	getNextRDPage := func() []models.Post {

		resp := r.getPosts(user.RedditUsername, user.RedditAuthToken)
		if resp.Err == nil {
			return resp.Posts
		} else {
			log.Printf("error getting rd page %v\n", resp.Err)
			return make([]models.Post, 0)
		}
	}

	return getNextRDPage, nil
}

func (r *Reddit) Name() string {
	return "Reddit"
}

func (r *Reddit) Weight() float64 {
	return r.weight
}

func (r *Reddit) getPosts(username, redditToken string) clients.PostResponse {
	posts := make([]models.Post, 0)

	// TODO: Eventually we will first have to check whether this token exists or if it expired
	if redditToken == "" {
		return clients.PostResponse{posts, "", fmt.Errorf("Unable to retrieve reddit oauth token from database for user: %v\n", username)}
	}

	client := &http.Client{}
	log.Printf("Token:%v\n", redditToken)
	jsonString := []byte(fmt.Sprintf("{ \"bearertoken\": \"%v\"}", redditToken))
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%v:%v/v1/%v/posts",
		r.Host, r.Port, username), bytes.NewBuffer(jsonString))
	if err != nil {
		return clients.PostResponse{posts, "", err}
	}

	redditResp, err := client.Do(req)
	if err != nil {
		return clients.PostResponse{posts, "", fmt.Errorf("Unable to get posts from reddit: %v", err)}
	}
	defer redditResp.Body.Close()

	err = json.NewDecoder(redditResp.Body).Decode(&posts)
	if err != nil {
		return clients.PostResponse{posts, "", fmt.Errorf("Unable to decode posts from Reddit: %v", err)}
	}

	log.Println("Successfully retrieved posts from reddit")
	return clients.PostResponse{posts, "", nil} // TODO: Don't return "", return the pagination URL
}
