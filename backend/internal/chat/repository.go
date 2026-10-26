package chat

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotImplemented = errors.New("chat repository: pendiente de implementar")

type Repository interface {
	ListConversations(ctx context.Context, userID string) ([]Conversation, error)
	FindOrCreateDirect(ctx context.Context, userID, peerID string) (Conversation, error)
	ListMessages(ctx context.Context, userID, conversationID, before string, limit int) (MessagePage, error)
	CreateMessage(ctx context.Context, userID, conversationID, clientMessageID, content string) (Message, error)
	IsMember(ctx context.Context, userID, conversationID string) (bool, error)
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) ListConversations(ctx context.Context, userID string) ([]Conversation, error) {
	return nil, ErrNotImplemented
}

func (r *PostgresRepository) FindOrCreateDirect(ctx context.Context, userID, peerID string) (Conversation, error) {
	return Conversation{}, ErrNotImplemented
}

func (r *PostgresRepository) ListMessages(ctx context.Context, userID, conversationID, before string, limit int) (MessagePage, error) {
	return MessagePage{}, ErrNotImplemented
}

func (r *PostgresRepository) CreateMessage(ctx context.Context, userID, conversationID, clientMessageID, content string) (Message, error) {
	return Message{}, ErrNotImplemented
}

func (r *PostgresRepository) IsMember(ctx context.Context, userID, conversationID string) (bool, error) {
	return false, ErrNotImplemented
}
