package rss

import (
	"encoding/json"
	"fmt"
	"strings"
	"log"
	"net/http"

	"github.com/iced-mocha/core/clients"
	"github.com/iced-mocha/shared/models"
)

var defaultFeeds = []string{
	"http://feeds.bbci.co.uk/news/rss.xml",
	"http://www.cbc.ca/cmlink/rss-topstories",
}

type RSS struct {
	Host   string
	Port   int
	weight float64
}

func New(host string, port int) *RSS {
	return &RSS{Host: host, Port: port}
}

func (r *RSS) GetPageGenerator(user *models.User) (func() []models.Post, error) {
	nextURL := fmt.Sprintf(
		"http://%v:%v/v1/posts?count=20&feeds=%v",
		r.Host,
		r.Port,
		strings.Join(defaultFeeds, ","))
	getNextPage := func() []models.Post {
		if nextURL == "" {
			return []models.Post{}
		}
		resp := r.getPosts(nextURL)
		if resp.Err == nil {
			nextURL = resp.NextURL
			return resp.Posts
		} else {
			nextURL = ""
			log.Printf("error getting rss page %v\n", resp.Err)
			return []models.Post{}
		}
	}
	return getNextPage, nil
}

func (r *RSS) Name() string {
	return "rss"
}

func (r *RSS) Weight() float64 {
	return r.weight
}

func (r *RSS) getPosts(url string) clients.PostResponse {
	rssPosts := make([]models.Post, 0)

	rssResp, err := http.Get(url)
	if err != nil {
		return clients.PostResponse{rssPosts, "", fmt.Errorf("Unable to fetch posts from rss: %v", err)}
	}
	defer rssResp.Body.Close()

	var rssRespBody models.ClientResp
	err = json.NewDecoder(rssResp.Body).Decode(&rssRespBody)
	if err != nil {
		return clients.PostResponse{rssPosts, "", fmt.Errorf("Unable to decode response from rss: %v", err)}
	}

	return clients.PostResponse{rssRespBody.Posts, rssRespBody.NextURL, nil}
}
