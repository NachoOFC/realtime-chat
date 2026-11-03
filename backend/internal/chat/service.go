package chat

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"strings"
	"unicode/utf8"
)

const maxMessageLength = 4000

var (
	ErrValidation = errors.New("datos inválidos")
	ErrForbidden  = errors.New("no perteneces a esta conversación")
)

type Service struct{ repository Repository }

func NewService(repository Repository) *Service { return &Service{repository: repository} }
func validID(value string) bool {
	id, err := uuid.Parse(value)
	return err == nil && id.String() == value && id != uuid.Nil
}
func (s *Service) ListConversations(ctx context.Context, userID string) ([]Conversation, error) {
	return s.repository.ListConversations(ctx, userID)
}
func (s *Service) CreateDirect(ctx context.Context, userID, peerID string) (Conversation, error) {
	peerID = strings.TrimSpace(peerID)
	if !validID(peerID) || peerID == userID {
		return Conversation{}, fmt.Errorf("%w: selecciona otro usuario válido", ErrValidation)
	}
	return s.repository.FindOrCreateDirect(ctx, userID, peerID)
}
func (s *Service) ListMessages(ctx context.Context, userID, conversationID, before string, limit int) (MessagePage, error) {
	if !validID(conversationID) {
		return MessagePage{}, fmt.Errorf("%w: conversación inválida", ErrValidation)
	}
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	return s.repository.ListMessages(ctx, userID, conversationID, before, limit)
}
func (s *Service) SendMessage(ctx context.Context, userID, conversationID, clientMessageID, content string) (Message, error) {
	content = strings.TrimSpace(content)
	if !validID(conversationID) || !validID(clientMessageID) || !utf8.ValidString(content) || content == "" || utf8.RuneCountInString(content) > maxMessageLength {
		return Message{}, fmt.Errorf("%w: el mensaje debe tener entre 1 y %d caracteres e identificadores válidos", ErrValidation, maxMessageLength)
	}
	return s.repository.CreateMessage(ctx, userID, conversationID, clientMessageID, content)
}
func (s *Service) IsMember(ctx context.Context, userID, conversationID string) (bool, error) {
	if !validID(userID) || !validID(conversationID) {
		return false, ErrValidation
	}
	return s.repository.IsMember(ctx, userID, conversationID)
}
func (s *Service) MemberIDs(ctx context.Context, userID, conversationID string) ([]string, error) {
	return s.repository.MemberIDs(ctx, userID, conversationID)
}
func (s *Service) MarkRead(ctx context.Context, userID, conversationID, messageID string) error {
	if !validID(conversationID) || !validID(messageID) {
		return ErrValidation
	}
	return s.repository.MarkRead(ctx, userID, conversationID, messageID)
}
