package domain

import "time"

type Profile struct {
	ID              int64
	Username        string
	Domain          string
	CreatedAt       time.Time
	IsOurSystemUser bool
	PasswordHash    string
	Salt            string
	AuthVersion     int
	Name            string
	Surname         string
	Patronymic      string
	Gender          string
	Birthday        time.Time
	AvatarPath      string
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
