package domain

type Sender struct {
	Id       int64  `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
}

func (s *Sender) SetAvatar(path string) {
	s.Avatar = path
}
