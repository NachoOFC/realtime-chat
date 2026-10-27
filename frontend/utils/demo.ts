import type { Conversation, Message, User } from '~/types'

const now = Date.now()
const minutesAgo = (minutes: number) => new Date(now - minutes * 60_000).toISOString()

export function isDemoMode(value: unknown) {
  return value === true || value === 'true' || value === '1'
}

export const demoUser: User = {
  id: 'demo-me',
  username: 'Tú',
  isGuest: true,
}

const lucia: User = { id: 'demo-lucia', username: 'Lucía', isGuest: true }
const martin: User = { id: 'demo-martin', username: 'Martín', isGuest: true }
const sofia: User = { id: 'demo-sofia', username: 'Sofía', isGuest: true }

const firstConversationMessages: Message[] = [
  {
    id: 'demo-m1',
    conversationId: 'demo-conv-1',
    senderId: lucia.id,
    content: 'Hola! Cómo va el proyecto?',
    createdAt: minutesAgo(45),
    status: 'read',
  },
  {
    id: 'demo-m2',
    conversationId: 'demo-conv-1',
    senderId: demoUser.id,
    content: 'Armando la interfaz del chat',
    createdAt: minutesAgo(43),
    status: 'read',
  },
  {
    id: 'demo-m3',
    conversationId: 'demo-conv-1',
    senderId: lucia.id,
    content: 'Se ve muy bien, estilo WhatsApp',
    createdAt: minutesAgo(40),
    status: 'read',
  },
  {
    id: 'demo-m4',
    conversationId: 'demo-conv-1',
    senderId: lucia.id,
    content: '¿Nos vemos mañana?',
    createdAt: minutesAgo(38),
    status: 'read',
  },
]

const secondConversationMessages: Message[] = [
  {
    id: 'demo-m5',
    conversationId: 'demo-conv-2',
    senderId: demoUser.id,
    content: 'Probaste el modo demo?',
    createdAt: minutesAgo(130),
    status: 'read',
  },
  {
    id: 'demo-m6',
    conversationId: 'demo-conv-2',
    senderId: martin.id,
    content: 'Sí, ya funciona',
    createdAt: minutesAgo(128),
    status: 'read',
  },
  {
    id: 'demo-m7',
    conversationId: 'demo-conv-2',
    senderId: martin.id,
    content: 'Jajaja buenísimo',
    createdAt: minutesAgo(125),
    status: 'read',
  },
]

const thirdConversationMessages: Message[] = [
  {
    id: 'demo-m8',
    conversationId: 'demo-conv-3',
    senderId: sofia.id,
    content: 'Te paso el archivo',
    createdAt: minutesAgo(1440),
    status: 'read',
  },
  {
    id: 'demo-m9',
    conversationId: 'demo-conv-3',
    senderId: sofia.id,
    content: 'Listo, revisalo cuando puedas',
    createdAt: minutesAgo(1435),
    status: 'read',
  },
]

export const demoConversations: Conversation[] = [
  {
    id: 'demo-conv-1',
    peer: lucia,
    lastMessage: firstConversationMessages[3],
    unreadCount: 2,
    updatedAt: firstConversationMessages[3]!.createdAt,
  },
  {
    id: 'demo-conv-2',
    peer: martin,
    lastMessage: secondConversationMessages[2],
    unreadCount: 0,
    updatedAt: secondConversationMessages[2]!.createdAt,
  },
  {
    id: 'demo-conv-3',
    peer: sofia,
    lastMessage: thirdConversationMessages[1],
    unreadCount: 1,
    updatedAt: thirdConversationMessages[1]!.createdAt,
  },
]

export const demoMessagesByConversation: Record<string, Message[]> = {
  'demo-conv-1': firstConversationMessages,
  'demo-conv-2': secondConversationMessages,
  'demo-conv-3': thirdConversationMessages,
}
