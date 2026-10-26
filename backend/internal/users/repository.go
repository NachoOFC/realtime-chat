package users

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound       = errors.New("usuario no encontrado")
	ErrNotImplemented = errors.New("users repository: pendiente de implementar")
)

type Repository interface {
	Create(ctx context.Context, email, username, passwordHash string) (User, error)
	FindByEmail(ctx context.Context, email string) (Credentials, error)
	FindByID(ctx context.Context, id string) (User, error)
	List(ctx context.Context, excludeID string) ([]User, error)
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, email, username, passwordHash string) (User, error) {
	return User{}, ErrNotImplemented
}

func (r *PostgresRepository) FindByEmail(ctx context.Context, email string) (Credentials, error) {
	return Credentials{}, ErrNotImplemented
}

func (r *PostgresRepository) FindByID(ctx context.Context, id string) (User, error) {
	return User{}, ErrNotImplemented
}

func (r *PostgresRepository) List(ctx context.Context, excludeID string) ([]User, error) {
	return nil, ErrNotImplemented
}
