package domain

import "time"

// Profile содержит расширенную информацию о пользователях почтовой системы.
type Profile struct {
	ID            int64     `json:"id" db:"id"`
	BaseProfileID int64     `json:"base_profile_id" db:"base_profile_id"`
	PasswordHash  string    `json:"-" db:"password_hash"`
	Salt          string    `json:"-" db:"salt"`
	Name          string    `json:"name" db:"name"`
	Surname       string    `json:"surname" db:"surname"`
	Patronymic    string    `json:"patronymic,omitempty" db:"patronymic"`
	Gender        string    `json:"gender,omitempty" db:"gender"`
	Birthday      time.Time `json:"birthday,omitempty" db:"birthday"`
	ImagePath     string    `json:"image_path,omitempty" db:"image_path"`
	AuthVersion   int       `json:"auth_version" db:"auth_version"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}
