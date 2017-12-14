package ranking

import (
	"github.com/iced-mocha/shared/models"
	"math"
	"math/rand"
	"time"
)

func getRank(p *models.Post, weight float64, sequenceLength int) float64 {
	age := time.Since(p.Date)
	ageMultiplier := math.Pow((age + time.Hour*12).Minutes(), 1.2)
	sequenceMultiplier := math.Pow(float64(1+sequenceLength), 0.2)
	randomMultiplier := rand.Float64()/10 + 1.0
	multiplier := ageMultiplier * sequenceMultiplier * randomMultiplier
	if multiplier <= 0 {
		return 0
	}
	return weight * (1.0 / multiplier)
}

func getNextProviderIndex(providers []*ContentProvider) int {
	topProvider := -1
	topRank := -1.0

	for i, p := range providers {
		if p.CurPost == nil || p.Weight == 0 {
			continue
		}

		curRank := getRank(p.CurPost, p.Weight, p.sequenceLength)
		if topProvider == -1 || curRank > topRank {
			topProvider = i
			topRank = curRank
		}
	}

	return topProvider
}

func resetSequenceLengths(providers []*ContentProvider) {
	for _, p := range providers {
		p.sequenceLength = 0
	}
}

// will modify the ContentProvider structs to contain the IDs of all viewed
// posts and the current page being looked at
func GetPosts(providers []*ContentProvider, count int) []models.Post {
	posts := make([]models.Post, 0)

	if len(providers) == 0 {
		return posts
	}

	for i := 0; i < count; i++ {
		p := getNextProviderIndex(providers)
		if p == -1 {
			break
		}
		provider := providers[p]
		posts = append(posts, *provider.CurPost)
		provider.NextPost()
		s := provider.sequenceLength + 1
		resetSequenceLengths(providers)
		provider.sequenceLength = s
	}

	return posts

}
