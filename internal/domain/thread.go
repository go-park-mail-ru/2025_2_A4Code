package domain

import "time"

//go:generate go run github.com/mailru/easyjson/easyjson@v0.9.1 -all thread.go

type Thread struct {
	ID          int64 `json:"id"`
	RootMessage int64 `json:"root_message"`
}

type ThreadInfo struct {
	ID           int64     `json:"id"`
	RootMessage  int64     `json:"root_message"`
	LastActivity time.Time `json:"last_activity"`
}
