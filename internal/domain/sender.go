package domain

//go:generate go run github.com/mailru/easyjson/easyjson@v0.9.1 -all sender.go

type Sender struct {
	Id       int64  `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
}

func (s *Sender) SetAvatar(path string) {
	s.Avatar = path
}
