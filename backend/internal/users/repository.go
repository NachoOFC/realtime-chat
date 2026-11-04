package users

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrEmailInUse     = errors.New("ya existe una cuenta con ese correo; inicia sesión con su método original")
	ErrNotFound       = errors.New("usuario no encontrado")
	ErrUsernameTaken  = errors.New("ese apodo ya está en uso")
	ErrNotImplemented = errors.New("users repository: pendiente de implementar")
)

const userColumns = `id::text, COALESCE(email, ''), username, COALESCE(avatar_url, ''), is_guest, last_seen_at, created_at, CASE WHEN google_subject IS NOT NULL THEN 'google' WHEN is_guest THEN 'guest' ELSE 'password' END`

type Repository interface {
	FindOrCreateGoogle(context.Context, GoogleIdentity) (User, error)
	IsActive(context.Context, string) (bool, error)
	DeleteGuest(context.Context, string, bool) (DeletedGuest, error)
	ExpiredGuestIDs(context.Context) ([]string, error)
	Create(ctx context.Context, email, username, passwordHash string) (User, error)
	CreateGuest(ctx context.Context, username string, expiresAt time.Time) (User, error)
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

func (r *PostgresRepository) CreateGuest(ctx context.Context, username string, expiresAt time.Time) (User, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO users (username, is_guest, guest_expires_at) VALUES ($1, true, $2) RETURNING `+userColumns,
		username, expiresAt,
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
		&user.AuthProvider,
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
		`SELECT `+userColumns+` FROM users WHERE id <> $1 AND (NOT is_guest OR guest_expires_at > now()) ORDER BY username LIMIT 100`,
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
		&user.AuthProvider,
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

func (r *PostgresRepository) FindOrCreateGoogle(ctx context.Context, identity GoogleIdentity) (User, error) {
	existing, err := scanUser(r.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE google_subject=$1`, identity.Subject))
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return User{}, err
	}
	base := normalizeUsername(identity.Name)
	base = strings.Map(func(c rune) rune {
		if usernamePattern.MatchString(string(c)) {
			return c
		}
		return -1
	}, base)
	base = normalizeUsername(base)
	if len([]rune(base)) < 3 {
		base = "Usuario"
	}
	if len([]rune(base)) > 20 {
		base = string([]rune(base)[:20])
	}
	for attempt := 0; attempt < 8; attempt++ {
		username := base
		if attempt > 0 {
			username = string([]rune(base)[:min(len([]rune(base)), 13)]) + "_" + uuid.NewString()[:6]
		}
		user, err := scanUser(r.pool.QueryRow(ctx, `INSERT INTO users (email,username,is_guest,google_subject,avatar_url)
  VALUES ($1,$2,false,$3,$4) ON CONFLICT (google_subject) DO UPDATE SET google_subject=EXCLUDED.google_subject RETURNING `+userColumns,
			strings.ToLower(strings.TrimSpace(identity.Email)), username, identity.Subject, identity.Picture))
		if err == nil {
			return user, nil
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if pgErr.ConstraintName == "users_email_key" {
				return User{}, ErrEmailInUse
			}
			continue
		}
		return User{}, err
	}
	return User{}, ErrUsernameTaken
}
func (r *PostgresRepository) IsActive(ctx context.Context, id string) (bool, error) {
	var active bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1 AND (NOT is_guest OR guest_expires_at>now()))`, id).Scan(&active)
	return active, err
}
func (r *PostgresRepository) ExpiredGuestIDs(ctx context.Context) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT id::text FROM users WHERE is_guest AND guest_expires_at<=now() ORDER BY guest_expires_at LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
func (r *PostgresRepository) DeleteGuest(ctx context.Context, id string, expiredOnly bool) (DeletedGuest, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return DeletedGuest{}, err
	}
	defer tx.Rollback(ctx)
	result := DeletedGuest{UserID: id, ConversationIDs: make([]string, 0), ParticipantIDs: make([]string, 0)}
	rows, err := tx.Query(ctx, `SELECT c.id::text FROM conversations c JOIN conversation_members cm ON cm.conversation_id=c.id
 WHERE cm.user_id=$1 ORDER BY c.id FOR UPDATE OF c`, id)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var cid string
		if err := rows.Scan(&cid); err != nil {
			rows.Close()
			return result, err
		}
		result.ConversationIDs = append(result.ConversationIDs, cid)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	rows, err = tx.Query(ctx, `SELECT DISTINCT user_id::text FROM conversation_members WHERE conversation_id=ANY($1::uuid[]) AND user_id<>$2`, result.ConversationIDs, id)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var peerID string
		if err := rows.Scan(&peerID); err != nil {
			rows.Close()
			return result, err
		}
		result.ParticipantIDs = append(result.ParticipantIDs, peerID)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	var deleted string
	err = tx.QueryRow(ctx, `DELETE FROM users WHERE id=$1 AND is_guest AND (NOT $2::boolean OR guest_expires_at<=now()) RETURNING id::text`, id, expiredOnly).Scan(&deleted)
	if errors.Is(err, pgx.ErrNoRows) {
		return result, ErrNotFound
	}
	if err != nil {
		return result, err
	}
	_, err = tx.Exec(ctx, `UPDATE conversation_members cm SET last_read_message_id=NULL WHERE conversation_id=ANY($1::uuid[])
 AND last_read_message_id IS NOT NULL AND NOT EXISTS(SELECT 1 FROM messages WHERE id=cm.last_read_message_id)`, result.ConversationIDs)
	if err != nil {
		return result, err
	}
	_, err = tx.Exec(ctx, `UPDATE conversation_members cm SET unread_count=(SELECT count(*) FROM messages m WHERE m.conversation_id=cm.conversation_id AND m.sender_id<>cm.user_id
 AND (cm.last_read_message_id IS NULL OR (m.created_at,m.id)>(SELECT created_at,id FROM messages WHERE id=cm.last_read_message_id)))
 WHERE conversation_id=ANY($1::uuid[])`, result.ConversationIDs)
	if err != nil {
		return result, err
	}
	_, err = tx.Exec(ctx, `DELETE FROM conversations c WHERE c.id=ANY($1::uuid[]) AND NOT EXISTS(SELECT 1 FROM conversation_members WHERE conversation_id=c.id)`, result.ConversationIDs)
	if err != nil {
		return result, err
	}
	if err = tx.Commit(ctx); err != nil {
		return result, err
	}
	return result, nil
}
