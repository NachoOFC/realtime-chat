import type { Conversation } from '~/types'

export function conversationTitle(conversation: Conversation): string {
  return conversation.kind === 'group' ? conversation.name || 'Grupo' : conversation.peer.id ? conversation.peer.username : 'Conversación finalizada'
}

export function canWriteConversation(conversation?: Conversation | null): boolean {
  return !!conversation && (conversation.kind === 'group' || !!conversation.peer.id)
}
