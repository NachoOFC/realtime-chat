package chat

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrClosed   = errors.New("el otro participante ya salió de la conversación")
	ErrNotFound = errors.New("usuario o mensaje no encontrado")
	ErrConflict = errors.New("clientMessageId ya utilizado para otro mensaje")
)

type Repository interface {
	ListConversations(context.Context, string) ([]Conversation, error)
	FindOrCreateDirect(context.Context, string, string) (Conversation, error)
	CreateGroup(context.Context, string, string, []string) (Conversation, error)
	AddGroupMembers(context.Context, string, string, []string) (Conversation, error)
	LeaveGroup(context.Context, string, string) ([]string, error)
	ListMessages(context.Context, string, string, string, int) (MessagePage, error)
	CreateMessage(context.Context, string, string, string, string) (Message, error)
	IsMember(context.Context, string, string) (bool, error)
	MemberIDs(context.Context, string, string) ([]string, error)
	MarkRead(context.Context, string, string, string) error
}

type PostgresRepository struct{ pool *pgxpool.Pool }

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

const conversationQuery = `SELECT c.id::text, COALESCE(peer.id::text, ''), COALESCE(peer.username, 'Conversación finalizada'),
 COALESCE(peer.avatar_url, ''), peer.last_seen_at, CASE WHEN peer.google_subject IS NOT NULL THEN 'google' WHEN peer.is_guest OR peer.id IS NULL THEN 'guest' ELSE 'password' END, mine.unread_count,
 COALESCE(last.created_at, c.created_at),
 CASE WHEN last.id IS NULL THEN NULL ELSE jsonb_build_object(
 'id', last.id, 'conversationId', last.conversation_id, 'senderId', last.sender_id,
 'content', last.content, 'clientMessageId', last.client_message_id, 'createdAt', last.created_at,
 'senderName', (SELECT username FROM users WHERE id=last.sender_id)) END,
 c.kind, COALESCE(c.name, ''),
 CASE WHEN c.kind='group' THEN (SELECT COALESCE(jsonb_agg(jsonb_build_object(
 'id', u.id, 'username', u.username, 'avatarUrl', COALESCE(u.avatar_url,''),
 'authProvider', CASE WHEN u.google_subject IS NOT NULL THEN 'google' WHEN u.is_guest THEN 'guest' ELSE 'password' END)
 ORDER BY lower(u.username), u.id), '[]'::jsonb)
 FROM conversation_members gm JOIN users u ON u.id=gm.user_id WHERE gm.conversation_id=c.id)
 ELSE '[]'::jsonb END
 FROM conversations c
 JOIN conversation_members mine ON mine.conversation_id = c.id AND mine.user_id = $1
 LEFT JOIN conversation_members other ON other.conversation_id = c.id AND other.user_id <> $1 AND c.kind='direct'
 LEFT JOIN users peer ON peer.id = other.user_id
 LEFT JOIN LATERAL (SELECT * FROM messages WHERE conversation_id = c.id
 ORDER BY created_at DESC, id DESC LIMIT 1) last ON true`

func scanConversation(row pgx.Row) (Conversation, error) {
	var c Conversation
	var last []byte
	var members []byte
	err := row.Scan(&c.ID, &c.Peer.ID, &c.Peer.Username, &c.Peer.AvatarURL, &c.Peer.LastSeenAt,
		&c.Peer.AuthProvider, &c.UnreadCount, &c.UpdatedAt, &last, &c.Kind, &c.Name, &members)
	if err == nil && last != nil {
		err = json.Unmarshal(last, &c.LastMessage)
	}
	if err == nil {
		err = json.Unmarshal(members, &c.Members)
	}
	return c, err
}

func (r *PostgresRepository) ListConversations(ctx context.Context, userID string) ([]Conversation, error) {
	rows, err := r.pool.Query(ctx, conversationQuery+` WHERE c.kind='group' OR peer.id IS NOT NULL OR last.id IS NOT NULL
 ORDER BY COALESCE(last.created_at, c.created_at) DESC, c.id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]Conversation, 0)
	for rows.Next() {
		c, err := scanConversation(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func (r *PostgresRepository) FindOrCreateDirect(ctx context.Context, userID, peerID string) (Conversation, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Conversation{}, err
	}
	defer tx.Rollback(ctx)
	var exists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`, peerID).Scan(&exists); err != nil {
		return Conversation{}, err
	}
	if !exists {
		return Conversation{}, ErrNotFound
	}
	ids := []string{userID, peerID}
	sort.Strings(ids)
	var id string
	// The unique key serializes concurrent attempts from either participant.
	err = tx.QueryRow(ctx, `INSERT INTO conversations (direct_key) VALUES ($1)
 ON CONFLICT (direct_key) DO UPDATE SET direct_key = EXCLUDED.direct_key RETURNING id::text`, strings.Join(ids, ":")).Scan(&id)
	if err != nil {
		return Conversation{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO conversation_members (conversation_id, user_id)
 VALUES ($1, $2), ($1, $3) ON CONFLICT DO NOTHING`, id, ids[0], ids[1])
	if err != nil {
		return Conversation{}, err
	}
	c, err := scanConversation(tx.QueryRow(ctx, conversationQuery+` WHERE c.id = $2`, userID, id))
	if err != nil {
		return Conversation{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Conversation{}, err
	}
	return c, nil
}

const messageColumns = `id::text, conversation_id::text, sender_id::text, content, client_message_id::text, created_at`

func scanMessage(row pgx.Row) (Message, error) {
	var m Message
	err := row.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Content, &m.ClientMessageID, &m.CreatedAt)
	return m, err
}

type messageCursor struct {
	At time.Time `json:"at"`
	ID string    `json:"id"`
}

func encodeCursor(m Message) string {
	data, _ := json.Marshal(messageCursor{At: m.CreatedAt, ID: m.ID})
	return base64.RawURLEncoding.EncodeToString(data)
}
func decodeCursor(value string) (messageCursor, error) {
	var c messageCursor
	data, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || json.Unmarshal(data, &c) != nil || c.At.IsZero() || uuid.Validate(c.ID) != nil {
		return c, fmt.Errorf("%w: cursor inválido", ErrValidation)
	}
	return c, nil
}
func (r *PostgresRepository) ListMessages(ctx context.Context, userID, conversationID, before string, limit int) (MessagePage, error) {
	var member, visible bool
	err := r.pool.QueryRow(ctx, `SELECT
 EXISTS(SELECT 1 FROM conversation_members WHERE conversation_id=$1 AND user_id=$2),
 EXISTS(SELECT 1 FROM conversation_members WHERE conversation_id=$1 AND user_id<>$2)
 OR EXISTS(SELECT 1 FROM messages WHERE conversation_id=$1)
 OR EXISTS(SELECT 1 FROM conversations WHERE id=$1 AND kind='group')`, conversationID, userID).Scan(&member, &visible)
	if err != nil {
		return MessagePage{}, err
	}
	if !member {
		return MessagePage{}, ErrForbidden
	}
	if !visible {
		return MessagePage{}, ErrNotFound
	}
	query := `SELECT ` + messageColumns + `,
 CASE WHEN EXISTS(SELECT 1 FROM conversation_members cm JOIN messages seen ON seen.id=cm.last_read_message_id
 WHERE cm.conversation_id=messages.conversation_id AND cm.user_id<>messages.sender_id
 AND (seen.created_at,seen.id)>=(messages.created_at,messages.id)) THEN 'read' ELSE 'sent' END,
 (SELECT username FROM users WHERE id=messages.sender_id)
 FROM messages WHERE conversation_id = $1`
	args := []any{conversationID, limit + 1}
	if before != "" {
		cursor, err := decodeCursor(before)
		if err != nil {
			return MessagePage{}, err
		}
		query += ` AND (created_at, id) < ($3, $4::uuid)`
		args = append(args, cursor.At, cursor.ID)
	}
	rows, err := r.pool.Query(ctx, query+` ORDER BY created_at DESC, id DESC LIMIT $2`, args...)
	if err != nil {
		return MessagePage{}, err
	}
	defer rows.Close()
	page := MessagePage{Messages: make([]Message, 0)}
	for rows.Next() {
		var m Message
		err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Content, &m.ClientMessageID, &m.CreatedAt, &m.Status, &m.SenderName)
		if err != nil {
			return MessagePage{}, err
		}
		page.Messages = append(page.Messages, m)
	}
	if err = rows.Err(); err != nil {
		return MessagePage{}, err
	}
	if len(page.Messages) > limit {
		page.Messages = page.Messages[:limit]
		page.NextCursor = encodeCursor(page.Messages[len(page.Messages)-1])
	}
	slices.Reverse(page.Messages)
	return page, nil
}

func lockConversation(ctx context.Context, tx pgx.Tx, userID, conversationID string) error {
	var id string
	err := tx.QueryRow(ctx, `SELECT c.id::text FROM conversations c WHERE c.id=$1
 AND EXISTS(SELECT 1 FROM conversation_members WHERE conversation_id=c.id AND user_id=$2)
 AND EXISTS(SELECT 1 FROM users actor WHERE actor.id=$2 AND (NOT actor.is_guest OR actor.guest_expires_at>now()))
 FOR UPDATE`, conversationID, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrForbidden
	}
	return err
}
func (r *PostgresRepository) CreateMessage(ctx context.Context, userID, conversationID, clientMessageID, content string) (Message, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Message{}, err
	}
	defer tx.Rollback(ctx)
	if err = lockConversation(ctx, tx, userID, conversationID); err != nil {
		return Message{}, err
	}
	var peerActive bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM conversations WHERE id=$1 AND kind='group') OR EXISTS(SELECT 1 FROM conversation_members cm JOIN users u ON u.id=cm.user_id WHERE cm.conversation_id=$1 AND cm.user_id<>$2 AND (NOT u.is_guest OR u.guest_expires_at>now()))`, conversationID, userID).Scan(&peerActive); err != nil {
		return Message{}, err
	}
	if !peerActive {
		return Message{}, ErrClosed
	}
	m, err := scanMessage(tx.QueryRow(ctx, `INSERT INTO messages (conversation_id, sender_id, client_message_id, content, created_at)
 VALUES ($1,$2,$3,$4,clock_timestamp()) ON CONFLICT (conversation_id, client_message_id) DO NOTHING RETURNING `+messageColumns,
		conversationID, userID, clientMessageID, content))
	if errors.Is(err, pgx.ErrNoRows) {
		m, err = scanMessage(tx.QueryRow(ctx, `SELECT `+messageColumns+` FROM messages WHERE conversation_id=$1 AND client_message_id=$2`, conversationID, clientMessageID))
		if err != nil {
			return Message{}, err
		}
		if m.SenderID != userID || m.Content != content {
			return Message{}, ErrConflict
		}
	} else if err != nil {
		return Message{}, err
	} else {
		_, err = tx.Exec(ctx, `UPDATE conversation_members SET unread_count=unread_count+1 WHERE conversation_id=$1 AND user_id<>$2`, conversationID, userID)
		if err != nil {
			return Message{}, err
		}
	}
	if err = tx.QueryRow(ctx, `SELECT username FROM users WHERE id=$1`, userID).Scan(&m.SenderName); err != nil {
		return Message{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Message{}, err
	}
	return m, nil
}
func (r *PostgresRepository) IsMember(ctx context.Context, userID, conversationID string) (bool, error) {
	var member bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM conversation_members WHERE conversation_id=$1 AND user_id=$2)`, conversationID, userID).Scan(&member)
	return member, err
}
func (r *PostgresRepository) MemberIDs(ctx context.Context, userID, conversationID string) ([]string, error) {
	member, err := r.IsMember(ctx, userID, conversationID)
	if err != nil {
		return nil, err
	}
	if !member {
		return nil, ErrForbidden
	}
	rows, err := r.pool.Query(ctx, `SELECT user_id::text FROM conversation_members WHERE conversation_id=$1`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]string, 0, 2)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	return result, rows.Err()
}
func (r *PostgresRepository) MarkRead(ctx context.Context, userID, conversationID, messageID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = lockConversation(ctx, tx, userID, conversationID); err != nil {
		return err
	}
	var at time.Time
	if err = tx.QueryRow(ctx, `SELECT created_at FROM messages WHERE id=$1 AND conversation_id=$2`, messageID, conversationID).Scan(&at); errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE conversation_members cm SET last_read_message_id=$3,
 unread_count=(SELECT count(*) FROM messages WHERE conversation_id=$1 AND sender_id<>$2 AND (created_at,id)>($4::timestamptz,$3::uuid))
 WHERE cm.conversation_id=$1 AND cm.user_id=$2 AND (cm.last_read_message_id IS NULL OR
 (SELECT (created_at,id) FROM messages WHERE id=cm.last_read_message_id) <= ($4::timestamptz,$3::uuid))`, conversationID, userID, messageID, at)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
