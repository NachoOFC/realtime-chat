package websocket

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/coder/websocket"
	"realtime-chat/backend/internal/auth"
	"realtime-chat/backend/internal/chat"
	"realtime-chat/backend/internal/httputil"
)

const maxMessageBytes = 32 * 1024

type Handler struct {
	hub           *Hub
	tickets       *TicketStore
	chat          *chat.Service
	allowedOrigin string
}

func NewHandler(hub *Hub, tickets *TicketStore, chatService *chat.Service, allowedOrigin string) *Handler {
	return &Handler{
		hub:           hub,
		tickets:       tickets,
		chat:          chatService,
		allowedOrigin: allowedOrigin,
	}
}

func (h *Handler) IssueTicket(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "No autenticado")
		return
	}

	ticket := h.tickets.Issue(userID)
	httputil.WriteJSON(w, http.StatusOK, map[string]string{"ticket": ticket})
}

func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.tickets.Consume(r.URL.Query().Get("ticket"))
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "Ticket inválido o expirado")
		return
	}

	conn, err := websocket.Accept(w, r, h.acceptOptions())
	if err != nil {
		slog.Warn("no se pudo aceptar websocket", "error", err)
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "bye")

	conn.SetReadLimit(maxMessageBytes)

	client := NewClient(userID, conn, h.hub)
	h.hub.Register(client)
	defer h.hub.Unregister(client)

	go client.WriteLoop(r.Context())
	client.ReadLoop(r.Context(), func(ctx context.Context, event InboundEvent) {
		h.handleEvent(ctx, client, event)
	})
}

func (h *Handler) handleEvent(ctx context.Context, client *Client, event InboundEvent) {
	switch event.Type {
	case EventConversationJoin, EventConversationLeave:
		if !h.requireMembership(ctx, client, event.ConversationID) {
			return
		}
		if event.Type == EventConversationJoin {
			h.hub.JoinConversation(client, event.ConversationID)
		} else {
			h.hub.LeaveConversation(client, event.ConversationID)
		}

	case EventMessageSend:
		message, err := h.chat.SendMessage(
			ctx,
			client.UserID,
			event.ConversationID,
			event.ClientMessageID,
			event.Content,
		)
		if err != nil {
			client.Send(OutboundEvent{Type: EventError, Error: "No se pudo enviar el mensaje"})
			return
		}

		client.Send(OutboundEvent{
			Type:            EventMessageAck,
			ConversationID:  message.ConversationID,
			ClientMessageID: event.ClientMessageID,
			Message:         &message,
		})
		h.hub.SendToConversation(message.ConversationID, OutboundEvent{
			Type:           EventMessageNew,
			ConversationID: message.ConversationID,
			Message:        &message,
		})

	case EventTyping:
		if !h.requireMembership(ctx, client, event.ConversationID) {
			return
		}
		h.hub.SendToConversation(event.ConversationID, OutboundEvent{
			Type:           EventTyping,
			ConversationID: event.ConversationID,
			UserID:         client.UserID,
			Active:         event.Active,
		})

	case EventMessageRead:
		if !h.requireMembership(ctx, client, event.ConversationID) {
			return
		}
		h.hub.SendToConversation(event.ConversationID, OutboundEvent{
			Type:           EventMessageRead,
			ConversationID: event.ConversationID,
			UserID:         client.UserID,
			MessageID:      event.MessageID,
		})

	default:
		client.Send(OutboundEvent{Type: EventError, Error: "Evento no soportado"})
	}
}

func (h *Handler) requireMembership(ctx context.Context, client *Client, conversationID string) bool {
	member, err := h.chat.IsMember(ctx, client.UserID, conversationID)
	if err != nil || !member {
		client.Send(OutboundEvent{Type: EventError, Error: "No perteneces a esta conversación"})
		return false
	}
	return true
}

func (h *Handler) acceptOptions() *websocket.AcceptOptions {
	if h.allowedOrigin == "*" {
		return &websocket.AcceptOptions{InsecureSkipVerify: true}
	}

	parsed, err := url.Parse(h.allowedOrigin)
	if err != nil || parsed.Host == "" {
		return &websocket.AcceptOptions{}
	}

	return &websocket.AcceptOptions{OriginPatterns: []string{parsed.Host}}
}
