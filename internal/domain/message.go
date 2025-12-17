package domain

import "time"

//go:generate go run github.com/mailru/easyjson/easyjson@v0.9.1 -all message.go

type Message struct {
	ID       string    `json:"id"`
	Topic    string    `json:"topic"`
	Snippet  string    `json:"snippet"`
	Datetime time.Time `json:"datetime"`
	IsRead   bool      `json:"is_read"`
	SenderID int64     `json:"sender_id"`
	Email    string    `json:"email"`
	Username string    `json:"username"`
	Avatar   string    `json:"avatar"`
}

type FullMessage struct {
	ID         string    `json:"id"`
	Topic      string    `json:"topic"`
	Text       string    `json:"text"`
	Datetime   time.Time `json:"datetime"`
	ThreadRoot string    `json:"thread_root"`
	FolderID   int64     `json:"folder_id"`
	FolderName string    `json:"folder_name"`
	SenderID   int64     `json:"sender_id"`
	Email      string    `json:"email"`
	Username   string    `json:"username"`
	Avatar     string    `json:"avatar"`
	Receivers  []string  `json:"receivers,omitempty"`
	Files      []File    `json:"files,omitempty"`
}

type Messages struct {
	MessageTotal  int         `json:"message_total"`
	MessageUnread int         `json:"message_unread"`
	MessageList   interface{} `json:"messages"`
}

type PaginatedMessages struct {
	NextCursor    string      `json:"next_cursor"`
	HasNext       bool        `json:"has_next"`
	MessageTotal  int         `json:"message_total"`
	MessageUnread int         `json:"message_unread"`
	MessageList   interface{} `json:"messages"`
}
