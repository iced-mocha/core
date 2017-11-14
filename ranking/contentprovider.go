package ranking

import (
	"github.com/iced-mocha/shared/models"
)

type ContentProvider struct {
	Weight   int
	CurPage  []models.Post
	NextPage func() []models.Post
	CurPost  *models.Post
	nextPost int
}

func NewContentProvider(weight int, nextPage func() []models.Post) *ContentProvider {
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
