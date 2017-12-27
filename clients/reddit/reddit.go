package reddit

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/iced-mocha/core/clients"
	"github.com/iced-mocha/shared/models"
)

type Reddit struct {
	Host   string
	Port   int
	weight float64
	client *http.Client
}

func New(host string, port int) *Reddit {
	// Only needed to test locally to allow for use of self signed certs
	cert, err := ioutil.ReadFile("/usr/local/etc/ssl/certs/reddit.crt")
	if err != nil {
		log.Fatal(err)
	}
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(cert)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: certPool,
			},
		},
	}
	return &Reddit{Host: host, Port: port, client: client}
}

func (r *Reddit) GetStartingURL(user models.User) string {
	if user.RedditUsername == "" {
		log.Printf("Unauthenticated user detected using default reddit page generator.")
		return fmt.Sprintf("https://%v:%v/v1/posts", r.Host, r.Port)
	}

	log.Printf("Getting reddit page generator for user: %v", user.Username)
	return fmt.Sprintf("https://%v:%v/v1/%v/posts", r.Host, r.Port, user.RedditUsername)
}

func (r *Reddit) GetDefaultPageGenerator() (func() []models.Post, error) {
	return r.GetPageGenerator(models.User{})
}

func (r *Reddit) GetPageGenerator(user models.User) (func() []models.Post, error) {
	nextURL := r.GetStartingURL(user)
	getNextPage := func() []models.Post {
		resp := r.getPosts(nextURL, user.RedditAuthToken, user.RedditRefreshToken)
		if resp.Err != nil {
			nextURL = ""
			log.Printf("Error getting reddit page %v", resp.Err)
			return []models.Post{}
		}

		// Next URL is specified by reddit client
		nextURL = resp.NextURL
		return resp.Posts
	}

	return getNextPage, nil
}

func (r *Reddit) Name() string {
	return "reddit"
}

func (r *Reddit) Weight() float64 {
	return r.weight
}

func (r *Reddit) getPosts(url, redditToken, refreshToken string) clients.PostResponse {
	posts := []models.Post{}

	// TODO Reddit-Clients GetPosts endpoint should accept request w/o these
	requestBody := []byte(fmt.Sprintf(`{"bearer-token": "%v", "refresh-token": "%v"}`, redditToken, refreshToken))

	req, err := http.NewRequest(http.MethodGet, url, bytes.NewBuffer(requestBody))
	if err != nil {
		return clients.PostResponse{posts, "", err}
	}

	log.Printf("Attemping to get reddit page with url: %v", url)
	resp, err := r.client.Do(req)
	if err != nil {
		return clients.PostResponse{posts, "", fmt.Errorf("Unable to get posts from reddit: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return clients.PostResponse{posts, "", fmt.Errorf("Unable to get posts from reddit received status code: %v", resp.StatusCode)}
	}

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return clients.PostResponse{posts, "", fmt.Errorf("Unable to get posts from reddit received status code: %v", resp.StatusCode)}
	}

	clientResp := models.ClientResp{}
	if json.Unmarshal(contents, &clientResp); err != nil {
		return clients.PostResponse{posts, "", fmt.Errorf("Unable to decode posts from Reddit: %v", err)}
	}
	log.Printf("Assigning next URL: %v", clientResp.NextURL)

	log.Println("Successfully retrieved posts from reddit")
	return clients.PostResponse{clientResp.Posts, clientResp.NextURL, nil}
}
