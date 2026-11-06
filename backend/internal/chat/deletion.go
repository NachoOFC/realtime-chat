package chat

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"realtime-chat/backend/internal/auth"
	"realtime-chat/backend/internal/httputil"
)

func (s *Service) DeleteMessage(ctx context.Context, actor, id, messageID string) ([]string, error) {
	if !validID(actor) || !validID(id) || !validID(messageID) {
		return nil, ErrValidation
	}
	return s.repository.DeleteMessage(ctx, actor, id, messageID)
}
func (s *Service) DeleteConversation(ctx context.Context, actor, id string) error {
	if !validID(actor) || !validID(id) {
		return ErrValidation
	}
	return s.repository.DeleteConversation(ctx, actor, id)
}

func (r *PostgresRepository) DeleteConversation(ctx context.Context, actor, id string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = lockConversation(ctx, tx, actor, id); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE conversation_members SET is_hidden=true,cleared_at=clock_timestamp(),unread_count=0 WHERE conversation_id=$1 AND user_id=$2`, id, actor)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PostgresRepository) DeleteMessage(ctx context.Context, actor, id, messageID string) ([]string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if err = lockConversation(ctx, tx, actor, id); err != nil {
		return nil, err
	}
	var owner string
	if err = tx.QueryRow(ctx, `SELECT sender_id::text FROM messages WHERE id=$1 AND conversation_id=$2`, messageID, id).Scan(&owner); errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	if owner != actor {
		return nil, ErrForbidden
	}
	_, err = tx.Exec(ctx, `UPDATE conversation_members cm SET last_read_message_id=(
 SELECT m.id FROM messages m WHERE m.conversation_id=$1 AND m.id<>$2
 AND (m.created_at,m.id)<=(SELECT created_at,id FROM messages WHERE id=$2)
 ORDER BY m.created_at DESC,m.id DESC LIMIT 1)
 WHERE cm.conversation_id=$1 AND cm.last_read_message_id=$2`, id, messageID)
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM messages WHERE id=$1 AND conversation_id=$2 AND sender_id=$3`, messageID, id, actor); err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `UPDATE conversation_members cm SET unread_count=(
 SELECT count(*) FROM messages m WHERE m.conversation_id=cm.conversation_id AND m.sender_id<>cm.user_id
 AND (cm.cleared_at IS NULL OR m.created_at>cm.cleared_at)
 AND (cm.last_read_message_id IS NULL OR (m.created_at,m.id)>(SELECT created_at,id FROM messages WHERE id=cm.last_read_message_id)))
 WHERE cm.conversation_id=$1`, id)
	if err != nil {
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
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return ids, nil
}

func (h *Handler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.UserID(r.Context())
	if !ok {
		httputil.Error(w, 401, "No autenticado")
		return
	}
	id, messageID := chi.URLParam(r, "id"), chi.URLParam(r, "messageId")
	ids, err := h.service.DeleteMessage(r.Context(), actor, id, messageID)
	if err != nil {
		writeError(w, err)
		return
	}
	if h.OnMessageDeleted != nil {
		h.OnMessageDeleted(id, messageID, ids)
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *Handler) DeleteConversation(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.UserID(r.Context())
	if !ok {
		httputil.Error(w, 401, "No autenticado")
		return
	}
	id := chi.URLParam(r, "id")
	if err := h.service.DeleteConversation(r.Context(), actor, id); err != nil {
		writeError(w, err)
		return
	}
	if h.OnChanged != nil {
		h.OnChanged(id, []string{actor}, "")
	}
	w.WriteHeader(http.StatusNoContent)
}
