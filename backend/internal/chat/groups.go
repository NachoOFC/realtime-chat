package chat

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

const maxGroupMembers = 50

func groupMemberIDs(actor string, ids []string) ([]string, error) {
	if len(ids) == 0 || len(ids) > maxGroupMembers-1 {
		return nil, fmt.Errorf("%w: selecciona entre 1 y 49 personas", ErrValidation)
	}
	seen := map[string]bool{actor: true}
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		if !validID(id) || seen[id] {
			return nil, fmt.Errorf("%w: participantes inválidos o repetidos", ErrValidation)
		}
		seen[id] = true
		result = append(result, id)
	}
	return result, nil
}

func (s *Service) CreateGroup(ctx context.Context, actor, name string, ids []string) (Conversation, error) {
	name = strings.TrimSpace(name)
	if !validID(actor) || !utf8.ValidString(name) || utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > 60 {
		return Conversation{}, fmt.Errorf("%w: el grupo necesita un nombre de hasta 60 caracteres", ErrValidation)
	}
	members, err := groupMemberIDs(actor, ids)
	if err != nil {
		return Conversation{}, err
	}
	return s.repository.CreateGroup(ctx, actor, name, members)
}

func (s *Service) AddGroupMembers(ctx context.Context, actor, id string, ids []string) (Conversation, error) {
	if !validID(actor) || !validID(id) {
		return Conversation{}, ErrValidation
	}
	members, err := groupMemberIDs(actor, ids)
	if err != nil {
		return Conversation{}, err
	}
	return s.repository.AddGroupMembers(ctx, actor, id, members)
}

func (s *Service) LeaveGroup(ctx context.Context, actor, id string) ([]string, error) {
	if !validID(actor) || !validID(id) {
		return nil, ErrValidation
	}
	return s.repository.LeaveGroup(ctx, actor, id)
}

func insertGroupMembers(ctx context.Context, tx pgx.Tx, id string, ids []string) error {
	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM users WHERE id=ANY($1::uuid[]) AND (NOT is_guest OR guest_expires_at>now())`, ids).Scan(&count); err != nil {
		return err
	}
	if count != len(ids) {
		return ErrNotFound
	}
	_, err := tx.Exec(ctx, `INSERT INTO conversation_members(conversation_id,user_id)
 SELECT $1,id FROM users WHERE id=ANY($2::uuid[]) ON CONFLICT DO NOTHING`, id, ids)
	return err
}

func (r *PostgresRepository) CreateGroup(ctx context.Context, actor, name string, ids []string) (Conversation, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Conversation{}, err
	}
	defer tx.Rollback(ctx)
	var id string
	if err = tx.QueryRow(ctx, `INSERT INTO conversations(kind,name) VALUES('group',$1) RETURNING id::text`, name).Scan(&id); err != nil {
		return Conversation{}, err
	}
	if err = insertGroupMembers(ctx, tx, id, append([]string{actor}, ids...)); err != nil {
		return Conversation{}, err
	}
	c, err := scanConversation(tx.QueryRow(ctx, conversationQuery+` WHERE c.id=$2`, actor, id))
	if err != nil {
		return Conversation{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Conversation{}, err
	}
	return c, nil
}

func lockGroup(ctx context.Context, tx pgx.Tx, actor, id string) error {
	if err := lockConversation(ctx, tx, actor, id); err != nil {
		return err
	}
	var kind string
	if err := tx.QueryRow(ctx, `SELECT kind FROM conversations WHERE id=$1`, id).Scan(&kind); err != nil {
		return err
	}
	if kind != "group" {
		return fmt.Errorf("%w: esta conversación no es un grupo", ErrValidation)
	}
	return nil
}

func (r *PostgresRepository) AddGroupMembers(ctx context.Context, actor, id string, ids []string) (Conversation, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Conversation{}, err
	}
	defer tx.Rollback(ctx)
	if err = lockGroup(ctx, tx, actor, id); err != nil {
		return Conversation{}, err
	}
	if err = insertGroupMembers(ctx, tx, id, ids); err != nil {
		return Conversation{}, err
	}
	var count int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM conversation_members WHERE conversation_id=$1`, id).Scan(&count); err != nil {
		return Conversation{}, err
	}
	if count > maxGroupMembers {
		return Conversation{}, fmt.Errorf("%w: el grupo admite hasta 50 participantes", ErrValidation)
	}
	c, err := scanConversation(tx.QueryRow(ctx, conversationQuery+` WHERE c.id=$2`, actor, id))
	if err != nil {
		return Conversation{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Conversation{}, err
	}
	return c, nil
}

func (r *PostgresRepository) LeaveGroup(ctx context.Context, actor, id string) ([]string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if err = lockGroup(ctx, tx, actor, id); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT user_id::text FROM conversation_members WHERE conversation_id=$1`, id)
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for rows.Next() {
		var member string
		if err = rows.Scan(&member); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, member)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM conversation_members WHERE conversation_id=$1 AND user_id=$2`, id, actor); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM conversations WHERE id=$1 AND NOT EXISTS(SELECT 1 FROM conversation_members WHERE conversation_id=$1)`, id); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return ids, nil
}
