package domain

import "time"

type Message struct {
	Id       string    `json:"id"`
	Sender   Sender    `json:"sender"`
	Subject  string    `json:"subject"`
	Snippet  string    `json:"snippet"`
	Datetime time.Time `json:"datetime"`
	IsRead   bool      `json:"is_read"`
	Folder   string    `json:"folder"`
}
