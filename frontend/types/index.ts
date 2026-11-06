export interface User {
  authProvider?: 'google' | 'guest' | 'password'
  id: string
  email?: string
  username: string
  avatarUrl?: string
  isGuest?: boolean
  lastSeenAt?: string
  createdAt?: string
}

export interface Message {
	senderName?: string
  id: string
  conversationId: string
  senderId: string
  content: string
  clientMessageId?: string
  createdAt: string
  status?: 'sending' | 'sent' | 'read'
}

export interface Conversation {
  kind?: 'direct' | 'group'
  name?: string
  members?: User[]
  id: string
  peer: User
  lastMessage?: Message
  unreadCount: number
  updatedAt: string
}

export interface MessagePage {
  messages: Message[]
  nextCursor?: string
}

export type ClientEvent =
  | {
      type: 'conversation.join' | 'conversation.leave'
      conversationId: string
    }
  | {
      type: 'message.send'
      conversationId: string
      clientMessageId: string
      content: string
    }
  | {
      type: 'typing'
      conversationId: string
      active: boolean
    }
  | {
      type: 'message.read'
      conversationId: string
      messageId: string
    }

export type ServerEvent =
  | { type: 'message.deleted'; conversationId: string; messageId: string }
  | { type: 'conversation.updated'; conversationId: string }
  | { type: 'user.removed'; userId: string }
  | { type: 'auth.expired' }
  | { type: 'message.new'; message: Message }
  | {
      type: 'message.ack'
      conversationId: string
      clientMessageId: string
      message: Message
    }
  | {
      type: 'typing'
      conversationId: string
      userId: string
      active: boolean
    }
  | {
      type: 'message.read'
      conversationId: string
      userId: string
      messageId: string
    }
  | { type: 'presence'; userId: string; online: boolean }
  | { type: 'error'; error: string }

export type SocketStatus = 'idle' | 'connecting' | 'open' | 'closed'
