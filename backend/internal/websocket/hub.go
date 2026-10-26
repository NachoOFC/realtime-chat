package websocket

import "sync"

type Hub struct {
	mu    sync.RWMutex
	users map[string]map[*Client]struct{}
	rooms map[string]map[*Client]struct{}
}

func NewHub() *Hub {
	return &Hub{
		users: make(map[string]map[*Client]struct{}),
		rooms: make(map[string]map[*Client]struct{}),
	}
}

func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	if h.users[client.UserID] == nil {
		h.users[client.UserID] = make(map[*Client]struct{})
	}
	h.users[client.UserID][client] = struct{}{}
	firstConnection := len(h.users[client.UserID]) == 1
	h.mu.Unlock()

	if firstConnection {
		h.Broadcast(OutboundEvent{
			Type:   EventPresence,
			UserID: client.UserID,
			Online: boolPtr(true),
		})
	}
}

func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()

	lastConnection := false
	if clients, ok := h.users[client.UserID]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.users, client.UserID)
			lastConnection = true
		}
	}

	for roomID, clients := range h.rooms {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.rooms, roomID)
		}
	}

	h.mu.Unlock()

	if lastConnection {
		h.Broadcast(OutboundEvent{
			Type:   EventPresence,
			UserID: client.UserID,
			Online: boolPtr(false),
		})
	}
}

func (h *Hub) JoinConversation(client *Client, conversationID string) {
	if conversationID == "" {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.rooms[conversationID] == nil {
		h.rooms[conversationID] = make(map[*Client]struct{})
	}
	h.rooms[conversationID][client] = struct{}{}
}

func (h *Hub) LeaveConversation(client *Client, conversationID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	clients, ok := h.rooms[conversationID]
	if !ok {
		return
	}

	delete(clients, client)
	if len(clients) == 0 {
		delete(h.rooms, conversationID)
	}
}

func (h *Hub) SendToConversation(conversationID string, event OutboundEvent) {
	h.mu.RLock()
	clients := make([]*Client, 0, len(h.rooms[conversationID]))
	for client := range h.rooms[conversationID] {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	for _, client := range clients {
		client.Send(event)
	}
}

func (h *Hub) SendToUser(userID string, event OutboundEvent) {
	h.mu.RLock()
	clients := make([]*Client, 0, len(h.users[userID]))
	for client := range h.users[userID] {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	for _, client := range clients {
		client.Send(event)
	}
}

func (h *Hub) Broadcast(event OutboundEvent) {
	h.mu.RLock()
	clients := make([]*Client, 0, len(h.users))
	for _, connections := range h.users {
		for client := range connections {
			clients = append(clients, client)
		}
	}
	h.mu.RUnlock()

	for _, client := range clients {
		client.Send(event)
	}
}

func (h *Hub) IsOnline(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.users[userID]) > 0
}

func boolPtr(value bool) *bool {
	return &value
}
