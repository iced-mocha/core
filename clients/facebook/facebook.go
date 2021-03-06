package facebook

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/iced-mocha/core/clients"
	"github.com/iced-mocha/shared/models"
)

type Facebook struct {
	Host   string
	Port   int
	weight float64
}

func New(host string, port int) *Facebook {
	return &Facebook{Host: host, Port: port}
}

func (f *Facebook) GetDefaultPageGenerator() (func() []models.Post, error) {
	defaultPosts := func() []models.Post {
		return []models.Post{}
	}

	return defaultPosts, nil
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

		resp := posts(nextFBURL)

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
	return "facebook"
}

func (f *Facebook) Weight() float64 {
	return f.weight
}

func posts(url string) clients.PostResponse {
	var fbRespBody models.ClientResp
	var fbPosts = make([]models.Post, 0)
	fbResp, err := http.Get(url)
	if err != nil {
		return clients.PostResponse{fbPosts, "", fmt.Errorf("Unable to get posts from facebook: %v", err)}
	}
	defer fbResp.Body.Close()

	err = json.NewDecoder(fbResp.Body).Decode(&fbRespBody)
	fbPosts = fbRespBody.Posts
	if err != nil {
		return clients.PostResponse{fbPosts, "", fmt.Errorf("Unable to decode posts from facebook: %v", err)}
	}

	return clients.PostResponse{fbPosts, fbRespBody.NextURL, nil}
}
