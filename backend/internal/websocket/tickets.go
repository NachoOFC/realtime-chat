package websocket

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type ticket struct {
	userID    string
	expiresAt time.Time
}

type TicketStore struct {
	mu      sync.Mutex
	tickets map[string]ticket
	ttl     time.Duration
}

func NewTicketStore(ttl time.Duration) *TicketStore {
	return &TicketStore{
		tickets: make(map[string]ticket),
		ttl:     ttl,
	}
}

func (s *TicketStore) Issue(userID string) string {
	now := time.Now()
	token := uuid.NewString()

	s.mu.Lock()
	defer s.mu.Unlock()

	for key, value := range s.tickets {
		if now.After(value.expiresAt) {
			delete(s.tickets, key)
		}
	}

	s.tickets[token] = ticket{
		userID:    userID,
		expiresAt: now.Add(s.ttl),
	}

	return token
}

func (s *TicketStore) Consume(token string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	value, ok := s.tickets[token]
	if !ok {
		return "", false
	}
	delete(s.tickets, token)

	if time.Now().After(value.expiresAt) {
		return "", false
	}

	return value.userID, true
}
