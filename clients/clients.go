package clients

import (
	"fmt"

	"github.com/iced-mocha/shared/models"
)

type Client interface {
	// returns a function which will return the next page of posts for the
	// content provider
	GetPageGenerator(user models.User) (func() []models.Post, error)
	GetDefaultPageGenerator() (func() []models.Post, error)
	Name() string
	Weight() float64
}

// Wrapper for the response from a post client
type PostResponse struct {
	Posts   []models.Post
	NextURL string
	Err     error
}

type InvalidAuth struct {
	ClientName   string
	ErrorMessage string
}

func (i InvalidAuth) Error() string {
	return fmt.Sprintf("Authentication error for %v: %v", i.ClientName, i.ErrorMessage)
}
