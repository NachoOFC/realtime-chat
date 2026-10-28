package users

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"realtime-chat/backend/internal/auth"
)

var usernamePattern = regexp.MustCompile(`^[\p{L}\p{N}_. -]+$`)

var (
	ErrInvalidCredentials = errors.New("email o contraseña incorrectos")
	ErrValidation         = errors.New("datos inválidos")
)

type Service struct {
	repository Repository
	jwtSecret  string
	tokenTTL   time.Duration
}

func NewService(repository Repository, jwtSecret string, tokenTTL time.Duration) *Service {
	return &Service{
		repository: repository,
		jwtSecret:  jwtSecret,
		tokenTTL:   tokenTTL,
	}
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (User, string, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))
	username := strings.TrimSpace(input.Username)

	if !strings.Contains(email, "@") {
		return User{}, "", fmt.Errorf("%w: email inválido", ErrValidation)
	}
	if len(username) < 3 || len(username) > 32 {
		return User{}, "", fmt.Errorf("%w: username debe tener entre 3 y 32 caracteres", ErrValidation)
	}
	if len(input.Password) < 8 {
		return User{}, "", fmt.Errorf("%w: password debe tener al menos 8 caracteres", ErrValidation)
	}

	passwordHash, err := auth.HashPassword(input.Password)
	if err != nil {
		return User{}, "", err
	}

	user, err := s.repository.Create(ctx, email, username, passwordHash)
	if err != nil {
		return User{}, "", err
	}

	token, err := auth.IssueToken(s.jwtSecret, user.ID, s.tokenTTL)
	if err != nil {
		return User{}, "", err
	}

	return user, token, nil
}

func (s *Service) GuestLogin(ctx context.Context, username string) (User, string, error) {
	username = normalizeUsername(username)

	switch {
	case utf8.RuneCountInString(username) < 3 || utf8.RuneCountInString(username) > 20:
		return User{}, "", fmt.Errorf("%w: el apodo debe tener entre 3 y 20 caracteres", ErrValidation)
	case !usernamePattern.MatchString(username):
		return User{}, "", fmt.Errorf("%w: el apodo solo puede tener letras, números, espacios, puntos, guiones y guiones bajos", ErrValidation)
	}

	user, err := s.repository.CreateGuest(ctx, username)
	if err != nil {
		return User{}, "", err
	}

	token, err := auth.IssueToken(s.jwtSecret, user.ID, s.tokenTTL)
	if err != nil {
		return User{}, "", err
	}

	return user, token, nil
}

func normalizeUsername(username string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(username)), " ")
}

func (s *Service) Login(ctx context.Context, input LoginInput) (User, string, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if email == "" || input.Password == "" {
		return User{}, "", fmt.Errorf("%w: email y password son obligatorios", ErrValidation)
	}

	credentials, err := s.repository.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return User{}, "", ErrInvalidCredentials
		}
		return User{}, "", err
	}

	if !auth.CheckPassword(credentials.PasswordHash, input.Password) {
		return User{}, "", ErrInvalidCredentials
	}

	token, err := auth.IssueToken(s.jwtSecret, credentials.User.ID, s.tokenTTL)
	if err != nil {
		return User{}, "", err
	}

	return credentials.User, token, nil
}

func (s *Service) FindByID(ctx context.Context, id string) (User, error) {
	return s.repository.FindByID(ctx, id)
}

func (s *Service) List(ctx context.Context, excludeID string) ([]User, error) {
	return s.repository.List(ctx, excludeID)
}
