package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/iced-mocha/shared/models"
)

type CoreHandler struct{}

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
