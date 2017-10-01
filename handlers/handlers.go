package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/iced-mocha/core/models"
)

type CoreHandler struct{}

func (api *CoreHandler) GetPosts(w http.ResponseWriter, r *http.Request) {

	// Testing begin
	var posts []models.Post
	samplePost := models.Post{"test1", time.Time{}, "test2", "test3", "test4", "test5", "test6", "test7"}
	for i := 0; i < 5; i++ {
		posts = append(posts, samplePost)
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
