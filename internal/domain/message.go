package domain

import "time"

type Message struct {
	ID       string    `json:"id"`
	Sender   Sender    `json:"sender"`
	Topic    string    `json:"topic"`
	Snippet  string    `json:"snippet"`
	Datetime time.Time `json:"datetime"`
	IsRead   bool      `json:"is_read"`
	Folder   string    `json:"folder"`
}
