package hackernews

import (
	"fmt"
	"log"
	"net/http"
	"encoding/json"

	"github.com/iced-mocha/shared/models"
	"github.com/iced-mocha/core/clients"
)

type HackerNews struct {
	Host string
	Port int
	weight float64
}

func New(Host string, Port int, Weight float64) *HackerNews {
	return &HackerNews{Host, Port, Weight}
}

func (h *HackerNews) GetPageGenerator(user models.User) (func() []models.Post, error) {
	nextURL := fmt.Sprintf("http://%v:%v/v1/posts?count=20", h.Host, h.Port)
	getNextPage := func() []models.Post {
		if nextURL == "" {
			return make([]models.Post, 0)
		}
		c := make(chan clients.PostResponse)
		go h.getPosts(c, nextURL)
		resp := <-c
		if resp.Err == nil {
			nextURL = resp.NextURL
			return resp.Posts
		} else {
			nextURL = ""
			log.Printf("error getting hn page %v\n", resp.Err)
			return make([]models.Post, 0)
		}
	}
	return getNextPage, nil
}

func (h *HackerNews) Name() string {
	return "Hacker News"
}

func (h *HackerNews) Weight() float64 {
	return h.weight
}

func (h *HackerNews) getPosts(c chan clients.PostResponse, url string) {
	hnPosts := make([]models.Post, 0)

	hnResp, err := http.Get(url)
	if err != nil {
		c <- clients.PostResponse{hnPosts, "", fmt.Errorf("Unable to fetch posts from hacker news: %v", err)}
		return
	}
	defer hnResp.Body.Close()

	var hnRespBody models.ClientResp
	err = json.NewDecoder(hnResp.Body).Decode(&hnRespBody)
	if err != nil {
		c <- clients.PostResponse{hnPosts, "", fmt.Errorf("Unable to decode response from hacker-news: %v", err)}
		return
	}

	log.Println("Successfully retrieved posts from hackernews")
	c <- clients.PostResponse{hnRespBody.Posts, hnRespBody.NextURL, nil}
}
