package websocket

import "realtime-chat/backend/internal/chat"

type EventType string

const (
	EventConversationJoin  EventType = "conversation.join"
	EventConversationLeave EventType = "conversation.leave"
	EventMessageSend       EventType = "message.send"
	EventMessageNew        EventType = "message.new"
	EventMessageAck        EventType = "message.ack"
	EventMessageRead       EventType = "message.read"
	EventTyping            EventType = "typing"
	EventPresence          EventType = "presence"
	EventError             EventType = "error"
)

type InboundEvent struct {
	Type            EventType `json:"type"`
	ConversationID  string    `json:"conversationId,omitempty"`
	ClientMessageID string    `json:"clientMessageId,omitempty"`
	MessageID       string    `json:"messageId,omitempty"`
	Content         string    `json:"content,omitempty"`
	Active          *bool     `json:"active,omitempty"`
}

type OutboundEvent struct {
	Type            EventType     `json:"type"`
	ConversationID  string        `json:"conversationId,omitempty"`
	UserID          string        `json:"userId,omitempty"`
	ClientMessageID string        `json:"clientMessageId,omitempty"`
	MessageID       string        `json:"messageId,omitempty"`
	Message         *chat.Message `json:"message,omitempty"`
	Active          *bool         `json:"active,omitempty"`
	Online          *bool         `json:"online,omitempty"`
	Error           string        `json:"error,omitempty"`
}
