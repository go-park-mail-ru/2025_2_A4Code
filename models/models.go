package models

import "time"

type BaseUser struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type RegisteredUser struct {
	BaseUser

	Username    string `json:"username"`
	DateOfBirth string `json:"date_of_birth"`
	Gender      string `json:"gender"`
}

type Sender struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
}

type Message struct {
	Id       string    `json:"id"`
	Sender   Sender    `json:"sender"`
	Subject  string    `json:"subject"`
	Snippet  string    `json:"snippet"`
	Datetime time.Time `json:"datetime"`
	IsRead   bool      `json:"is_read"`
	Folder   string    `json:"folder"`
}
