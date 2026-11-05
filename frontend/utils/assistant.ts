import type { Message } from '~/types'

export const ASSISTANT_CONVERSATION_ID = 'iachat'
export const ASSISTANT_SENDER_ID = 'assistant'
export const MAX_ASSISTANT_USER_RUNES = 4000
export const MAX_ASSISTANT_HISTORY_MESSAGES = 21
export const MAX_ASSISTANT_HISTORY_RUNES = 16000

export interface AssistantTurn {
  role: 'user' | 'assistant'
  content: string
}

export function runeLength(text: string): number {
  return Array.from(text).length
}

export function buildAssistantHistory(messages: Message[], senderId: string, content: string): AssistantTurn[] {
  const history: AssistantTurn[] = []
  let pendingUser: string | undefined

  for (const message of messages) {
    if (message.conversationId !== ASSISTANT_CONVERSATION_ID || message.status === 'sending') continue
    if (message.senderId === senderId) {
      pendingUser = message.content
    } else if (message.senderId === ASSISTANT_SENDER_ID && pendingUser !== undefined) {
      history.push({ role: 'user', content: pendingUser }, { role: 'assistant', content: message.content })
      pendingUser = undefined
    }
  }

  history.push({ role: 'user', content })
  let totalRunes = history.reduce((total, turn) => total + runeLength(turn.content), 0)
  while (history.length > MAX_ASSISTANT_HISTORY_MESSAGES || totalRunes > MAX_ASSISTANT_HISTORY_RUNES) {
    if (history.length < 3) break
    const removed = history.splice(0, 2)
    totalRunes -= removed.reduce((total, turn) => total + runeLength(turn.content), 0)
  }
  return history
}
