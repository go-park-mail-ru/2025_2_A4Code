package domain

import "time"

type Message struct {
	//ID       string    `json:"id"`
	Sender   Sender    `json:"sender"`
	Topic    string    `json:"topic"`
	Snippet  string    `json:"snippet"`
	Datetime time.Time `json:"datetime"`
	IsRead   bool      `json:"is_read"`
	Folder   string    `json:"folder"`
}

type FullMessage struct {
	Topic    string    `json:"topic"`
	Text     string    `json:"text"`
	Datetime time.Time `json:"datetime"`
	Folder
	Sender
	Thread
	Files
}

type Messages struct {
	MessageTotal  int       `json:"message_total"`
	MessageUnread int       `json:"message_unread"`
	Messages      []Message `json:"messages"`
}
