package domain

type RegisteredUser struct {
	BaseUser

	Username    string `json:"username"`
	DateOfBirth string `json:"date_of_birth"`
	Gender      string `json:"gender"`
}
