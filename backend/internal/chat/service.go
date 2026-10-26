package chat

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const maxMessageLength = 4000

var (
	ErrValidation = errors.New("datos inválidos")
	ErrForbidden  = errors.New("no perteneces a esta conversación")
)

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) ListConversations(ctx context.Context, userID string) ([]Conversation, error) {
	return s.repository.ListConversations(ctx, userID)
}

func (s *Service) CreateDirect(ctx context.Context, userID, peerID string) (Conversation, error) {
	peerID = strings.TrimSpace(peerID)
	if peerID == "" || peerID == userID {
		return Conversation{}, fmt.Errorf("%w: peerId inválido", ErrValidation)
	}

	return s.repository.FindOrCreateDirect(ctx, userID, peerID)
}

func (s *Service) ListMessages(ctx context.Context, userID, conversationID, before string, limit int) (MessagePage, error) {
	if conversationID == "" {
		return MessagePage{}, fmt.Errorf("%w: conversación inválida", ErrValidation)
	}
	if limit <= 0 || limit > 100 {
		limit = 30
	}

	return s.repository.ListMessages(ctx, userID, conversationID, before, limit)
}

func (s *Service) SendMessage(ctx context.Context, userID, conversationID, clientMessageID, content string) (Message, error) {
	content = strings.TrimSpace(content)

	switch {
	case conversationID == "" || clientMessageID == "":
		return Message{}, fmt.Errorf("%w: conversación o clientMessageId inválido", ErrValidation)
	case content == "":
		return Message{}, fmt.Errorf("%w: el mensaje está vacío", ErrValidation)
	case len(content) > maxMessageLength:
		return Message{}, fmt.Errorf("%w: el mensaje supera %d caracteres", ErrValidation, maxMessageLength)
	}

	member, err := s.repository.IsMember(ctx, userID, conversationID)
	if err != nil {
		return Message{}, err
	}
	if !member {
		return Message{}, ErrForbidden
	}

	return s.repository.CreateMessage(ctx, userID, conversationID, clientMessageID, content)
}

func (s *Service) IsMember(ctx context.Context, userID, conversationID string) (bool, error) {
	if userID == "" || conversationID == "" {
		return false, fmt.Errorf("%w: usuario o conversación inválidos", ErrValidation)
	}
	return s.repository.IsMember(ctx, userID, conversationID)
}
