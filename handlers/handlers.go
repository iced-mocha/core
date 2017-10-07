package handlers

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/iced-mocha/core/storage/driver"
	"github.com/iced-mocha/core/storage/driver/sqlite"
	"github.com/iced-mocha/shared/models"
)

type CoreHandler struct{}

var d driver.StorageDriver

func init() {
	var err error

	d, err = sqlite.New(sqlite.Config{})
	if err != nil {
		log.Printf("Unable to create driver: %v\n", err)
	}
}

func (api *CoreHandler) InsertUser(w http.ResponseWriter, r *http.Request) {
	// Read body of the request
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}

	// Change the body into a user object
	user := &models.User{}
	err = json.Unmarshal(body, user)
	if err != nil {
		log.Printf("Error parsing body: %v", err)
		http.Error(w, "can't parse body", http.StatusBadRequest)
		return
	}

	userId := d.InsertUser(user.Name)
	log.Println("userId: " + userId)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(userId))

}

func (api *CoreHandler) GetPosts(w http.ResponseWriter, r *http.Request) {
	// Testing begin
	posts := make([]models.Post, 0)
	hnResp, err := http.Get("http://hacker-news:4000/v1/posts?count=20")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewDecoder(hnResp.Body).Decode(&posts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Testing end

	res, err := json.Marshal(posts)
	if err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write(res)
	} else {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
