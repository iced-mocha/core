package ranking

import (
	"github.com/iced-mocha/shared/models"
)

func getNextNonEmpty(providers []*ContentProvider, start int) int {
	if len(providers) == 0 {
		return -1
	}

	if start >= len(providers) {
		start = 0
	}

	i := start
	for {
		if i >= len(providers) {
			i = 0
		}

		if providers[i].CurPost != nil {
			return i
		}

		i++
		if i == start {
			break
		}
	}

	return -1
}

// will modify the ContentProvider structs to contain the IDs of all viewed
// posts and the current page being looked at
func GetPosts(providers []*ContentProvider, count int) []models.Post {
	posts := make([]models.Post, 0)

	if len(providers) == 0 {
		return posts
	}

	// TODO: Do this more intelligently
	p := 0
	for i := 0; i < count; i++ {
		p = getNextNonEmpty(providers, p)
		if p == -1 {
			break
		}
		provider := providers[p]
		posts = append(posts, *provider.CurPost)
		provider.NextPost()
		p++
	}

	return posts

}
