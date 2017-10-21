package comparators

import (
	"time"

	"github.com/iced-mocha/shared/models"
)

type ByPostRank struct {
	Posts []models.Post
	// default weight is 1, higher weight moves posts lower
	PlatformWeights map[string]float64
}

func (s ByPostRank) Len() int {
	return len(s.Posts)
}

func (s ByPostRank) Swap(i, j int) {
	a := s.Posts
	a[i], a[j] = a[j], a[i]
}

func (s ByPostRank) Less(i, j int) bool {
	return s.getRank(i) < s.getRank(j)
}

func (s ByPostRank) getRank(i int) float64 {
	p := s.Posts[i]
	age := time.Since(p.Date)
	weight := 1.0
	if v, ok := s.PlatformWeights[p.Platform]; ok {
		weight = v
	}

	dayDuration := time.Duration(24) * time.Hour
	// a 1 day old post will have double the rank of a new post
	return float64(i) * weight * (age.Minutes() + dayDuration.Minutes())
}
