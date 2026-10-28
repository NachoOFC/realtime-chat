package users

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound       = errors.New("usuario no encontrado")
	ErrUsernameTaken  = errors.New("ese apodo ya está en uso")
	ErrNotImplemented = errors.New("users repository: pendiente de implementar")
)

const userColumns = `id::text, COALESCE(email, ''), username, COALESCE(avatar_url, ''), is_guest, last_seen_at, created_at`

type Repository interface {
	Create(ctx context.Context, email, username, passwordHash string) (User, error)
	CreateGuest(ctx context.Context, username string) (User, error)
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
	row := r.pool.QueryRow(ctx,
		`INSERT INTO users (email, username, password_hash) VALUES ($1, $2, $3) RETURNING `+userColumns,
		email, username, passwordHash,
	)

	user, err := scanUser(row)
	if isUniqueViolation(err) {
		return User{}, ErrUsernameTaken
	}
	return user, err
}

func (r *PostgresRepository) CreateGuest(ctx context.Context, username string) (User, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO users (username, is_guest) VALUES ($1, true) RETURNING `+userColumns,
		username,
	)

	user, err := scanUser(row)
	if isUniqueViolation(err) {
		return User{}, ErrUsernameTaken
	}
	return user, err
}

func (r *PostgresRepository) FindByEmail(ctx context.Context, email string) (Credentials, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+userColumns+`, password_hash FROM users WHERE lower(email) = lower($1)`,
		email,
	)

	var (
		user         User
		passwordHash *string
	)

	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.AvatarURL,
		&user.IsGuest,
		&user.LastSeenAt,
		&user.CreatedAt,
		&passwordHash,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Credentials{}, ErrNotFound
	}
	if err != nil {
		return Credentials{}, err
	}

	credentials := Credentials{User: user}
	if passwordHash != nil {
		credentials.PasswordHash = *passwordHash
	}
	return credentials, nil
}

func (r *PostgresRepository) FindByID(ctx context.Context, id string) (User, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id)
	return scanUser(row)
}

func (r *PostgresRepository) List(ctx context.Context, excludeID string) ([]User, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+userColumns+` FROM users WHERE id <> $1 ORDER BY username LIMIT 100`,
		excludeID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]User, 0)
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, rows.Err()
}

func scanUser(row pgx.Row) (User, error) {
	var user User
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.AvatarURL,
		&user.IsGuest,
		&user.LastSeenAt,
		&user.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return user, err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
