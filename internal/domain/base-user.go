package domain

type BaseUser struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}
