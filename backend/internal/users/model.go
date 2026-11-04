package users

import "time"

type User struct {
	AuthProvider string     `json:"authProvider"`
	ID           string     `json:"id"`
	Email        string     `json:"email,omitempty"`
	Username     string     `json:"username"`
	AvatarURL    string     `json:"avatarUrl,omitempty"`
	IsGuest      bool       `json:"isGuest"`
	LastSeenAt   *time.Time `json:"lastSeenAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
}

type GuestInput struct {
	Username string `json:"username"`
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

// GoogleIdentity is built only after verifying Google's signed ID token.
type GoogleIdentity struct{ Subject, Email, Name, Picture string }
type DeletedGuest struct {
	UserID          string
	ConversationIDs []string
	ParticipantIDs  []string
}
