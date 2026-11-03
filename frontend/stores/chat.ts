import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import type { Conversation, Message, MessagePage, User } from '~/types'
import { useApi } from '~/composables/useApi'

export const useChatStore = defineStore('chat', () => {
  const conversations = ref<Conversation[]>([])
  const users = ref<User[]>([])
  const onlineUsers = ref<Record<string, boolean>>({})
  const activeConversationId = ref<string | null>(null)
  const messages = ref<Record<string, Message[]>>({})
  const loadedHistories = ref<Record<string, boolean>>({})
  const nextCursors = ref<Record<string, string | undefined>>({})
  const loadingMessages = ref<Record<string, boolean>>({})
  const typingByConversation = ref<Record<string, string[]>>({})
  const loadingConversations = ref(false)
  const loadingUsers = ref(false)
  const errorMessage = ref<string | null>(null)
  const activeConversation = computed(() => conversations.value.find(item => item.id === activeConversationId.value) ?? null)
  const activeMessages = computed(() => messages.value[activeConversationId.value ?? ''] ?? [])
  const activeTypingUsers = computed(() => typingByConversation.value[activeConversationId.value ?? ''] ?? [])

  async function loadConversations() {
    loadingConversations.value = true
    try { conversations.value = await useApi<Conversation[]>('/api/conversations') }
    catch { errorMessage.value = 'No se pudieron cargar las conversaciones.' }
    finally { loadingConversations.value = false }
  }
  async function loadUsers() {
    loadingUsers.value = true
    try { users.value = await useApi<User[]>('/api/users') }
    catch { errorMessage.value = 'No se pudieron cargar los usuarios.' }
    finally { loadingUsers.value = false }
  }
  async function startConversation(peerId: string) {
    try {
      const conversation = await useApi<Conversation>('/api/conversations', { method: 'POST', body: { peerId } })
      if (!conversations.value.some(item => item.id === conversation.id)) conversations.value.unshift(conversation)
      await selectConversation(conversation.id)
      return true
    } catch { errorMessage.value = 'No se pudo iniciar la conversación. Intenta de nuevo.'; return false }
  }
  async function selectConversation(id: string) {
    activeConversationId.value = id
    if (!loadedHistories.value[id]) await loadMessages(id)
  }
  async function loadMessages(id: string, before?: string) {
    if (loadingMessages.value[id]) return
    loadingMessages.value[id] = true
    try {
      const page = await useApi<MessagePage>(`/api/conversations/${id}/messages`, { query: { before } })
      const current = messages.value[id] ?? []
      const combined = before ? [...page.messages, ...current] : [...page.messages, ...current.filter(item => !page.messages.some(m => m.id === item.id || (m.clientMessageId && m.clientMessageId === item.clientMessageId)))]
      messages.value[id] = combined.filter((item, index) => combined.findIndex(m => m.id === item.id) === index)
      loadedHistories.value[id] = true
      nextCursors.value[id] = page.nextCursor
    } catch { errorMessage.value = 'No se pudo cargar el historial.' }
    finally { loadingMessages.value[id] = false }
  }
  function upsertMessage(message: Message) {
    const current = messages.value[message.conversationId] ?? []
    const index = current.findIndex(item => item.id === message.id || (message.clientMessageId && item.clientMessageId === message.clientMessageId))
    if (index >= 0) {
      current[index] = { ...message, status: current[index]?.status === 'read' ? 'read' : message.status }
      return false
    }
    messages.value[message.conversationId] = [...current, message]
    return true
  }
  function confirmMessage(clientMessageId: string, message: Message) { upsertMessage({ ...message, clientMessageId, status: 'sent' }) }
  function setTyping(conversationId: string, userId: string, active: boolean) {
    const current = typingByConversation.value[conversationId] ?? []
    typingByConversation.value[conversationId] = active ? Array.from(new Set([...current, userId])) : current.filter(id => id !== userId)
  }
  function updateConversationPreview(message: Message) {
    const index = conversations.value.findIndex(item => item.id === message.conversationId)
    if (index < 0) { void loadConversations(); return }
    const conversation = conversations.value[index]!
    if (conversation.lastMessage && conversation.lastMessage.createdAt > message.createdAt) return
    conversations.value.splice(index, 1)
    conversations.value.unshift({ ...conversation, lastMessage: message, updatedAt: message.createdAt })
  }
  function markRead(conversationId: string, userId: string, messageId: string) {
    const current = messages.value[conversationId] ?? []
    const index = current.findIndex(item => item.id === messageId)
    if (index >= 0) {
      for (const item of current.slice(0, index + 1)) if (item.senderId !== userId) item.status = 'read'
    }
  }
  function reset() {
    conversations.value = []; users.value = []; onlineUsers.value = {}; activeConversationId.value = null
    messages.value = {}; loadedHistories.value = {}; nextCursors.value = {}; loadingMessages.value = {}; typingByConversation.value = {}; errorMessage.value = null
  }
  function clearError() { errorMessage.value = null }
  return { conversations, users, onlineUsers, activeConversationId, messages, nextCursors, loadingMessages, typingByConversation,
    loadingConversations, loadingUsers, errorMessage, activeConversation, activeMessages, activeTypingUsers,
    loadConversations, loadUsers, startConversation, selectConversation, loadMessages, upsertMessage, confirmMessage,
    setTyping, updateConversationPreview, markRead, reset, clearError }
})
