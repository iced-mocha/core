package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/iced-mocha/shared/models"
)

type CoreHandler struct{}

func (api *CoreHandler) GetPosts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	// TODO: Abstract this, make it concurrent
	hnPosts := make([]models.Post, 0)
	hnResp, err := http.Get("http://hacker-news-client:4000/v1/posts?count=20")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewDecoder(hnResp.Body).Decode(&hnPosts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var fbId string
	var fbToken string
	if v, ok := query["fb_id"]; ok && len(v) != 0 {
		fbId = v[0]
	}
	if v, ok := query["fb_token"]; ok && len(v) != 0 {
		fbToken = v[0]
	}
	fbPosts := make([]models.Post, 0)
	fbResp, err := http.Get("http://facebook-client:5000/v1/posts?fb_id="+fbId+"&fb_token="+fbToken)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewDecoder(fbResp.Body).Decode(&fbPosts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	posts := append(hnPosts, fbPosts...)

	res, err := json.Marshal(posts)
	if err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write(res)
	} else {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
