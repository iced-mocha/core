package facebook

import (
	"log"
	"fmt"
	"encoding/json"
	"net/http"

	"github.com/iced-mocha/core/clients"
	"github.com/iced-mocha/shared/models"
)

type Facebook struct {
	Host string
	Port int
	weight float64
}

func New(Host string, Port int, Weight float64) *Facebook {
	return &Facebook{Host, Port, Weight}
}

func (f *Facebook) GetPageGenerator(user models.User) (func() []models.Post, error) {
	if user.FacebookAuthToken == "" {
		return nil, clients.InvalidAuth{f.Name(), "empty auth token"}
	}

	nextFBURL := fmt.Sprintf("http://%v:%v/v1/posts?fb_token=%v", f.Host, f.Port, user.FacebookAuthToken)
	getNextFBPage := func() []models.Post {
		if nextFBURL == "" {
			return make([]models.Post, 0)
		}

		c := make(chan clients.PostResponse)
		go posts(nextFBURL, c)
		resp := <-c

		if resp.Err == nil {
			nextFBURL = resp.NextURL
			return resp.Posts
		} else {
			log.Printf("error getting fb page %v\n", resp.Err)
			return make([]models.Post, 0)
		}
	}

	return getNextFBPage, nil
}

func (f *Facebook) Name() string {
	return "Facebook"
}

func (f *Facebook) Weight() float64 {
	return f.weight
}

func posts(url string, c chan clients.PostResponse) {
	var fbRespBody models.ClientResp
	var fbPosts = make([]models.Post, 0)
	fbResp, err := http.Get(url)
	if err != nil {
		c <- clients.PostResponse{fbPosts, "", fmt.Errorf("Unable to get posts from facebook: %v", err)}
		return
	}
	defer fbResp.Body.Close()

	err = json.NewDecoder(fbResp.Body).Decode(&fbRespBody)
	fbPosts = fbRespBody.Posts
	if err != nil {
		c <- clients.PostResponse{fbPosts, "", fmt.Errorf("Unable to decode posts from facebook: %v", err)}
		return
	}

	c <- clients.PostResponse{fbPosts, fbRespBody.NextURL, nil}
}
