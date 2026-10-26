package users

import "time"

type User struct {
	ID         string     `json:"id"`
	Email      string     `json:"email"`
	Username   string     `json:"username"`
	AvatarURL  string     `json:"avatarUrl,omitempty"`
	LastSeenAt *time.Time `json:"lastSeenAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
}

type RegisterInput struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Credentials struct {
	User         User
	PasswordHash string
}
