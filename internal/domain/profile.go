package domain

import "time"

type Profile struct {
	ID              int64     `json:"id"`
	Username        string    `json:"username"`
	Domain          string    `json:"domain"`
	CreatedAt       time.Time `json:"created_at"`
	IsOurSystemUser bool      `json:"is_our_system_user"`
	PasswordHash    string    `json:"password_hash"`
	AuthVersion     int       `json:"auth_version"`
	Name            string    `json:"name"`
	Surname         string    `json:"surname"`
	Patronymic      string    `json:"patronymic"`
	Gender          string    `json:"gender"`
	Birthday        time.Time `json:"birthday"`
	AvatarPath      string    `json:"avatar_path"`
	Settings
}

type ProfileInfo struct {
	Username   string    `json:"username"`
	CreatedAt  time.Time `json:"created_at"`
	Name       string    `json:"name"`
	Surname    string    `json:"surname"`
	Patronymic string    `json:"patronymic"`
	Gender     string    `json:"gender"`
	Birthday   string    `json:"birthday"`
	AvatarPath string    `json:"avatar_path"`
	Settings
}

func (p *Profile) Email() string {
	return p.Username + "@" + p.Domain
}

func (p *Profile) DisplayName() string {
	if p.Name != "" && p.Surname != "" && p.Patronymic != "" {
		return p.Name + " " + p.Surname + " " + p.Patronymic
	}
	if p.Name != "" && p.Surname != "" {
		return p.Name + " " + p.Surname
	}
	if p.Name != "" {
		return p.Name
	}
	return p.Username
}
