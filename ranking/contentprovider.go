package ranking

import (
	"github.com/iced-mocha/shared/models"
)

// a structure to keep track of the posts being read from a single content
// provider for a single user. It records which page we are on, and contains
// a function used to get the next page of data from the content provider
type ContentProvider struct {
	Weight         float64
	CurPage        []models.Post
	NextPage       func() []models.Post
	CurPost        *models.Post
	nextPageChan   chan []models.Post
	nextPost       int
	sequenceLength int
}

func NewContentProvider(weight float64, nextPage func() []models.Post) *ContentProvider {
	c := &ContentProvider{
		Weight:       weight,
		NextPage:     nextPage,
		nextPageChan: make(chan []models.Post, 1),
	}
	c.NextPost()
	return c
}

func (c *ContentProvider) NextPost() {
	// preload the next page if we are getting close to needing it
	if c.nextPost == len(c.CurPage)/2 {
		go func() {
			c.nextPageChan <- c.NextPage()
		}()
	}

	if c.nextPost >= len(c.CurPage) {
		c.CurPage = <-c.nextPageChan
		c.nextPost = 0
		if len(c.CurPage) == 0 {
			c.CurPost = nil
			return
		}
	}

	c.CurPost = &c.CurPage[c.nextPost]
	c.nextPost++
}
