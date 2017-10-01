package models

import (
	"time"
)

type Post struct {
	ID         string
	Date       time.Time
	Author     string
	ProfileImg string
	Title      string
	Content    string
	HeroImg    string
	PostLink   string
}
