package domain

import "time"

type Thread struct {
	ID          int64
	RootMessage int64
}

type ThreadInfo struct {
	ID           int64
	RootMessage  int64
	LastActivity time.Time
}
