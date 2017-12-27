package twitter

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

type Twitter struct {
	Host   string
	Port   int
	weight float64
	client *http.Client
}

func New(host string, port int) *Twitter {
	// Load reddit-clients certifiate so we know we can trust reddit-client
	caCert, err := ioutil.ReadFile("/usr/local/etc/ssl/certs/twitter.crt")
	if err != nil {
		log.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		},
	}
	return &Twitter{Host: host, Port: port, client: client}
}

// We currently do not support unauthenticated twitter posts
func (t *Twitter) GetDefaultPageGenerator() (func() []models.Post, error) {
	defaultPosts := func() []models.Post {
		return []models.Post{}
	}

	return defaultPosts, nil
}

func (t *Twitter) GetPageGenerator(user models.User) (func() []models.Post, error) {
	if user.TwitterUsername == "" {
		log.Printf("Getting unauthenicated twitter page generator.")
		//nextURL = fmt.Sprintf("https://%v:%v/v1/posts", r.Host, r.Port)
		// TODO: Currently dont support this so
		return (func() []models.Post { return []models.Post{} }), nil
	}

	log.Printf("Getting twitter page generator for user: %v", user.Username)
	nextURL := fmt.Sprintf("https://%v:%v/v1/%v/posts", t.Host, t.Port, user.Username)

	getNextPage := func() []models.Post {
		log.Printf("Attemping to get twitter page with url: %v", nextURL)
		resp := t.getPosts(nextURL, user.TwitterAuthToken, user.TwitterSecret)
		if resp.Err != nil {
			nextURL = ""
			log.Printf("Error getting twitter page %v", resp.Err)
			return []models.Post{}
		}

		nextURL = resp.NextURL
		return resp.Posts
	}

	return getNextPage, nil
}

func (t *Twitter) Name() string {
	return "twitter"
}

func (t *Twitter) Weight() float64 {
	return t.weight
}

func (t *Twitter) getPosts(url, token, secret string) clients.PostResponse {
	posts := []models.Post{}

	authData := []byte(fmt.Sprintf(`{ "token": "%v", "secret": "%v"}`, token, secret))
	req, err := http.NewRequest(http.MethodGet, url, bytes.NewBuffer(authData))
	if err != nil {
		return clients.PostResponse{posts, "", err}
	}

	log.Printf("About to get twitter posts for URL %v", url)
	resp, err := t.client.Do(req)
	if err != nil {
		return clients.PostResponse{posts, "", fmt.Errorf("Unable to get posts from twitter: %v", err)}
	}
	defer resp.Body.Close()

	log.Printf("Received status code %v requesting posts from %v", resp.StatusCode, url)
	if resp.StatusCode != http.StatusOK {
		return clients.PostResponse{posts, "", fmt.Errorf("Unable to get posts from twitter received status code: %v", resp.StatusCode)}
	}

	clientResp := models.ClientResp{}
	err = json.NewDecoder(resp.Body).Decode(&clientResp)
	if err != nil {
		return clients.PostResponse{clientResp.Posts, "", fmt.Errorf("Unable to decode posts from Twitter: %v", err)}
	}
	posts = clientResp.Posts

	log.Printf("Successfully retrieved %v posts from twitter", len(posts))
	// Note here I am treating the `nextURL` really as a URI
	return clients.PostResponse{posts, fmt.Sprintf("https://%v:%v%v", t.Host, t.Port, clientResp.NextURL), nil}
}
