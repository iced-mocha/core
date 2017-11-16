package ranking

import (
	"github.com/iced-mocha/shared/models"
)

type ContentProvider struct {
	Weight         float64
	CurPage        []models.Post
	NextPage       func() []models.Post
	CurPost        *models.Post
	nextPost       int
	sequenceLength int
}

func NewContentProvider(weight float64, nextPage func() []models.Post) *ContentProvider {
	c := &ContentProvider{Weight: weight, NextPage: nextPage}
	c.NextPost()
	return c
}

func (c *ContentProvider) NextPost() {
	if c.nextPost >= len(c.CurPage) {
		c.CurPage = c.NextPage()
		c.nextPost = 0
		if len(c.CurPage) == 0 {
			c.CurPost = nil
			return
		}
	}

	c.CurPost = &c.CurPage[c.nextPost]
	c.nextPost++
}
