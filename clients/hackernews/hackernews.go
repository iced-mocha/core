package hackernews

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/iced-mocha/core/clients"
	"github.com/iced-mocha/shared/models"
)

type HackerNews struct {
	Host   string
	Port   int
	weight float64
}

func New(Host string, Port int, Weight float64) *HackerNews {
	return &HackerNews{Host, Port, Weight}
}

func (h *HackerNews) GetPageGenerator(user *models.User) (func() []models.Post, error) {
	nextURL := fmt.Sprintf("http://%v:%v/v1/posts?count=20", h.Host, h.Port)
	getNextPage := func() []models.Post {
		if nextURL == "" {
			return []models.Post{}
		}
		resp := h.getPosts(nextURL)
		if resp.Err == nil {
			nextURL = resp.NextURL
			return resp.Posts
		} else {
			nextURL = ""
			log.Printf("error getting hn page %v\n", resp.Err)
			return []models.Post{}
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

func (h *HackerNews) getPosts(url string) clients.PostResponse {
	hnPosts := make([]models.Post, 0)

	hnResp, err := http.Get(url)
	if err != nil {
		return clients.PostResponse{hnPosts, "", fmt.Errorf("Unable to fetch posts from hacker news: %v", err)}
	}
	defer hnResp.Body.Close()

	var hnRespBody models.ClientResp
	err = json.NewDecoder(hnResp.Body).Decode(&hnRespBody)
	if err != nil {
		return clients.PostResponse{hnPosts, "", fmt.Errorf("Unable to decode response from hacker-news: %v", err)}
	}

	return clients.PostResponse{hnRespBody.Posts, hnRespBody.NextURL, nil}
}
