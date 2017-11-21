package googlenews

import (
	"log"
	"fmt"
	"net/http"
	"encoding/json"

	"github.com/iced-mocha/shared/models"
	"github.com/iced-mocha/core/clients"
)

type GoogleNews struct {
	Host string
	Port int
	weight float64
}

func New(Host string, Port int, Weight float64) *GoogleNews {
	return &GoogleNews{Host, Port, Weight}
}

func (g *GoogleNews) GetPageGenerator(user models.User) (func() []models.Post, error) {
	called := false
	getNextPage := func() []models.Post {
		// google news is not paginated, so if we have gotten the first page,
		// we have gotten all the pages
		if called {
			return make([]models.Post, 0)
		}
		called = true

		c := make(chan clients.PostResponse)
		go g.posts(c)
		resp := <-c
		if resp.Err == nil {
			return resp.Posts
		} else {
			log.Printf("error getting gn page %v\n", resp.Err)
			return make([]models.Post, 0)
		}
	}
	return getNextPage, nil
}

func (g *GoogleNews) Name() string {
	return "Google News"
}

func (g *GoogleNews) Weight() float64 {
	return g.weight
}

func (g *GoogleNews) posts(c chan clients.PostResponse) {
	gnPosts := make([]models.Post, 0, 0)

	gnResp, err := http.Get(fmt.Sprintf("http://%v:%v/v1/posts?count=20", g.Host, g.Port))
	if err != nil {
		c <- clients.PostResponse{gnPosts, "", fmt.Errorf("Unable to fetch posts from google news: %v", err)}
		return
	}
	defer gnResp.Body.Close()

	err = json.NewDecoder(gnResp.Body).Decode(&gnPosts)
	if err != nil {
		c <- clients.PostResponse{gnPosts, "", fmt.Errorf("Unable to decode response from google-news: %v", err)}
		return
	}

	log.Println("Successfully retrieved posts from googlenews")
	c <- clients.PostResponse{gnPosts, "", nil}
}
