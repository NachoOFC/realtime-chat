package chat

import "time"

type Peer struct {
	AuthProvider string     `json:"authProvider"`
	ID           string     `json:"id"`
	Username     string     `json:"username"`
	AvatarURL    string     `json:"avatarUrl,omitempty"`
	LastSeenAt   *time.Time `json:"lastSeenAt,omitempty"`
}

type Conversation struct {
	Kind        string    `json:"kind"`
	Name        string    `json:"name,omitempty"`
	Members     []Peer    `json:"members,omitempty"`
	ID          string    `json:"id"`
	Peer        Peer      `json:"peer"`
	LastMessage *Message  `json:"lastMessage,omitempty"`
	UnreadCount int       `json:"unreadCount"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Message struct {
	SenderName      string    `json:"senderName,omitempty"`
	Status          string    `json:"status,omitempty"`
	ID              string    `json:"id"`
	ConversationID  string    `json:"conversationId"`
	SenderID        string    `json:"senderId"`
	Content         string    `json:"content"`
	ClientMessageID string    `json:"clientMessageId"`
	CreatedAt       time.Time `json:"createdAt"`
}

type MessagePage struct {
	Messages   []Message `json:"messages"`
	NextCursor string    `json:"nextCursor,omitempty"`
}

type CreateConversationInput struct {
	PeerID string `json:"peerId"`
}

type CreateGroupInput struct {
	Name      string   `json:"name"`
	MemberIDs []string `json:"memberIds"`
}

type AddGroupMembersInput struct {
	MemberIDs []string `json:"memberIds"`
}
