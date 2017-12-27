package googlenews

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/iced-mocha/core/clients"
	"github.com/iced-mocha/shared/models"
)

type GoogleNews struct {
	Host   string
	Port   int
	weight float64
}

func New(host string, port int) *GoogleNews {
	return &GoogleNews{Host: host, Port: port}
}

func (g *GoogleNews) GetDefaultPageGenerator() (func() []models.Post, error) {
	called := false
	getNextPage := func() []models.Post {
		// google news is not paginated, so if we have gotten the first page,
		// we have gotten all the pages
		if called {
			return make([]models.Post, 0)
		}
		called = true

		resp := g.posts()
		if resp.Err == nil {
			return resp.Posts
		} else {
			log.Printf("error getting gn page %v\n", resp.Err)
			return make([]models.Post, 0)
		}
	}
	return getNextPage, nil
}

func (g *GoogleNews) GetPageGenerator(user models.User) (func() []models.Post, error) {
	return g.GetDefaultPageGenerator()
}

func (g *GoogleNews) Name() string {
	return "google-news"
}

func (g *GoogleNews) Weight() float64 {
	return g.weight
}

func (g *GoogleNews) posts() clients.PostResponse {
	gnPosts := make([]models.Post, 0, 0)

	gnResp, err := http.Get(fmt.Sprintf("http://%v:%v/v1/posts?count=20", g.Host, g.Port))
	if err != nil {
		return clients.PostResponse{gnPosts, "", fmt.Errorf("Unable to fetch posts from google news: %v", err)}
	}
	defer gnResp.Body.Close()

	err = json.NewDecoder(gnResp.Body).Decode(&gnPosts)
	if err != nil {
		return clients.PostResponse{gnPosts, "", fmt.Errorf("Unable to decode response from google-news: %v", err)}
	}

	log.Println("Successfully retrieved posts from googlenews")
	return clients.PostResponse{gnPosts, "", nil}
}
