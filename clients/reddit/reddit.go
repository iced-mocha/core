package reddit

import (
	"fmt"
	"log"
	"bytes"
	"net/http"
	"encoding/json"

	"github.com/iced-mocha/shared/models"
	"github.com/iced-mocha/core/clients"
)

type Reddit struct {
	Host string
	Port int
	weight float64
}

func New(Host string, Port int, Weight float64) *Reddit {
	return &Reddit{Host, Port, Weight}
}

func (r *Reddit) GetPageGenerator(user models.User) (func() []models.Post, error) {
	getNextRDPage := func() []models.Post {
		// TODO: should get next page, not same page over and over
		c := make(chan clients.PostResponse)

		go r.getPosts(c, user.RedditUsername, user.RedditAuthToken)
		resp := <-c
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

func (r *Reddit) getPosts(c chan clients.PostResponse, username, redditToken string) {
	posts := make([]models.Post, 0)

	// TODO: Eventually we will first have to check whether this token exists or if it expired
	if redditToken == "" {
		c <- clients.PostResponse{posts, "", fmt.Errorf("Unable to retrieve reddit oauth token from database for user: %v\n", username)}
		return
	}

	client := &http.Client{}
	log.Printf("Token:%v\n", redditToken)
	jsonString := []byte(fmt.Sprintf("{ \"bearertoken\": \"%v\"}", redditToken))
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%v:%v/v1/%v/posts",
		r.Host, r.Port, username), bytes.NewBuffer(jsonString))
	if err != nil {
		c <- clients.PostResponse{posts, "", err}
		return
	}

	redditResp, err := client.Do(req)
	if err != nil {
		c <- clients.PostResponse{posts, "", fmt.Errorf("Unable to get posts from reddit: %v", err)}
		return
	}
	defer redditResp.Body.Close()

	err = json.NewDecoder(redditResp.Body).Decode(&posts)
	if err != nil {
		c <- clients.PostResponse{posts, "", fmt.Errorf("Unable to decode posts from Reddit: %v", err)}
		return
	}

	log.Println("Successfully retrieved posts from reddit")
	c <- clients.PostResponse{posts, "", nil} // TODO: Don't return "", return the pagination URL
}
