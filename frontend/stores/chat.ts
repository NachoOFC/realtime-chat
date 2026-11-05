import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import type { Conversation, Message, MessagePage, User } from '~/types'
import { useApi } from '~/composables/useApi'
import { useAssistantStore } from '~/stores/assistant'
import { ASSISTANT_CONVERSATION_ID } from '~/utils/assistant'

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
  let sessionEpoch = 0
  let historyEpoch = 0
  let conversationsRequest = 0
  let usersRequest = 0
  let conversationEpochs: Record<string, number> = {}
  const activeConversation = computed(() => conversations.value.find(item => item.id === activeConversationId.value) ?? null)
  const activeMessages = computed(() => messages.value[activeConversationId.value ?? ''] ?? [])
  const activeTypingUsers = computed(() => typingByConversation.value[activeConversationId.value ?? ''] ?? [])

  async function loadConversations() {
    const epoch = sessionEpoch
    const request = ++conversationsRequest
    loadingConversations.value = true
    try {
      const result = await useApi<Conversation[]>('/api/conversations')
      if (epoch === sessionEpoch && request === conversationsRequest) {
        const visible = new Set(result.map(item => item.id))
        for (const previous of conversations.value) if (!visible.has(previous.id)) discardConversation(previous.id)
        conversations.value = result
        const selected = activeConversationId.value
        if (selected && selected !== ASSISTANT_CONVERSATION_ID && !visible.has(selected)) discardConversation(selected)
      }
    }
    catch { if (epoch === sessionEpoch && request === conversationsRequest) errorMessage.value = 'No se pudieron cargar las conversaciones.' }
    finally { if (epoch === sessionEpoch && request === conversationsRequest) loadingConversations.value = false }
  }
  async function loadUsers() {
    const epoch = sessionEpoch
    const request = ++usersRequest
    loadingUsers.value = true
    try {
      const result = await useApi<User[]>('/api/users')
      if (epoch === sessionEpoch && request === usersRequest) users.value = result
    }
    catch { if (epoch === sessionEpoch && request === usersRequest) errorMessage.value = 'No se pudieron cargar los usuarios.' }
    finally { if (epoch === sessionEpoch && request === usersRequest) loadingUsers.value = false }
  }
  async function startConversation(peerId: string) {
    const epoch = sessionEpoch
    try {
      const conversation = await useApi<Conversation>('/api/conversations', { method: 'POST', body: { peerId } })
      if (epoch !== sessionEpoch) return false
      // An older list response must not remove a conversation we just created.
      conversationsRequest++
      loadingConversations.value = false
      if (!conversations.value.some(item => item.id === conversation.id)) conversations.value.unshift(conversation)
      await selectConversation(conversation.id)
      return epoch === sessionEpoch
    } catch { if (epoch === sessionEpoch) errorMessage.value = 'No se pudo iniciar la conversación. Intenta de nuevo.'; return false }
  }
  async function selectConversation(id: string) {
    activeConversationId.value = id
    if (id === ASSISTANT_CONVERSATION_ID) return
    if (!loadedHistories.value[id]) await loadMessages(id)
  }
  function replaceGroup(conversation: Conversation) {
    conversationsRequest++
    loadingConversations.value = false
    const index = conversations.value.findIndex(item => item.id === conversation.id)
    if (index < 0) conversations.value.unshift(conversation)
    else conversations.value[index] = conversation
  }
  function groupError(error: unknown, fallback: string) {
    const message = (error as { data?: { error?: string } })?.data?.error
    return typeof message === 'string' ? message : fallback
  }
  async function createGroup(name: string, memberIds: string[]) {
    const epoch = sessionEpoch
    try {
      const group = await useApi<Conversation>('/api/conversations/groups', { method: 'POST', body: { name, memberIds } })
      if (epoch !== sessionEpoch) return false
      replaceGroup(group)
      await selectConversation(group.id)
      return epoch === sessionEpoch
    } catch (error) { if (epoch === sessionEpoch) errorMessage.value = groupError(error, 'No se pudo crear el grupo.'); return false }
  }
  async function addGroupMembers(id: string, memberIds: string[]) {
    const epoch = sessionEpoch
    try {
      const group = await useApi<Conversation>(`/api/conversations/${id}/members`, { method: 'POST', body: { memberIds } })
      if (epoch !== sessionEpoch) return false
      replaceGroup(group)
      return true
    } catch (error) { if (epoch === sessionEpoch) errorMessage.value = groupError(error, 'No se pudieron agregar las personas.'); return false }
  }
  async function leaveGroup(id: string) {
    const epoch = sessionEpoch
    try {
      await useApi(`/api/conversations/${id}/members/me`, { method: 'DELETE' })
      if (epoch !== sessionEpoch) return false
      conversationsRequest++
      loadingConversations.value = false
      conversations.value = conversations.value.filter(item => item.id !== id)
      discardConversation(id)
      return true
    } catch (error) { if (epoch === sessionEpoch) errorMessage.value = groupError(error, 'No se pudo salir del grupo.'); return false }
  }
  async function loadMessages(id: string, before?: string) {
    if (id === ASSISTANT_CONVERSATION_ID) return
    if (loadingMessages.value[id]) return
    const epoch = sessionEpoch
    const history = historyEpoch
    const conversationEpoch = conversationEpochs[id] ?? 0
    const initialIDs = new Set((messages.value[id] ?? []).map(message => message.id))
    loadingMessages.value[id] = true
    try {
      const page = await useApi<MessagePage>(`/api/conversations/${id}/messages`, { query: { before } })
      if (epoch !== sessionEpoch || history !== historyEpoch || conversationEpoch !== (conversationEpochs[id] ?? 0)) return
      const current = messages.value[id] ?? []
      // The first page replaces confirmed cache entries, including deleted messages.
      // Preserve optimistic sends and events that arrived after this request began.
      const preserved = before ? current : current.filter(item => item.status === 'sending' || !initialIDs.has(item.id))
      const combined = [...page.messages, ...preserved]
      const result: Message[] = []
      for (const message of combined) {
        const existing = result.find(item => item.id === message.id || (message.clientMessageId && item.clientMessageId === message.clientMessageId))
        if (!existing) result.push(message)
        else if (message.status === 'read') existing.status = 'read'
      }
      messages.value[id] = result
      loadedHistories.value[id] = true
      nextCursors.value[id] = page.nextCursor
    } catch (error) {
      if (epoch !== sessionEpoch || history !== historyEpoch || conversationEpoch !== (conversationEpochs[id] ?? 0)) return
      const status = (error as { statusCode?: number; status?: number }).statusCode ?? (error as { status?: number }).status
      if (status === 403 || status === 404 || status === 410) {
        conversations.value = conversations.value.filter(item => item.id !== id)
        discardConversation(id)
      } else errorMessage.value = 'No se pudo cargar el historial.'
    }
    finally { if (epoch === sessionEpoch && history === historyEpoch && conversationEpoch === (conversationEpochs[id] ?? 0)) loadingMessages.value[id] = false }
  }
  function discardConversation(id: string) {
    conversationEpochs[id] = (conversationEpochs[id] ?? 0) + 1
    delete messages.value[id]
    delete loadedHistories.value[id]
    delete nextCursors.value[id]
    delete loadingMessages.value[id]
    delete typingByConversation.value[id]
    if (activeConversationId.value === id) activeConversationId.value = null
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
  function invalidateHistories() {
    historyEpoch++
    const pending: Record<string, Message[]> = {}
    for (const [id, current] of Object.entries(messages.value)) {
      const sending = current.filter(message => message.status === 'sending')
      if (sending.length) pending[id] = sending
    }
    messages.value = pending
    loadedHistories.value = {}
    nextCursors.value = {}
    loadingMessages.value = {}
  }
  function reset() {
    useAssistantStore().reset()
    sessionEpoch++
    historyEpoch++
    conversationEpochs = {}
    conversations.value = []; users.value = []; onlineUsers.value = {}; activeConversationId.value = null
    messages.value = {}; loadedHistories.value = {}; nextCursors.value = {}; loadingMessages.value = {}; typingByConversation.value = {}; errorMessage.value = null
    loadingConversations.value = false
    loadingUsers.value = false
  }
  function clearError() { errorMessage.value = null }
  return { conversations, users, onlineUsers, activeConversationId, messages, nextCursors, loadingMessages, typingByConversation,
    loadingConversations, loadingUsers, errorMessage, activeConversation, activeMessages, activeTypingUsers,
    loadConversations, loadUsers, startConversation, createGroup, addGroupMembers, leaveGroup, selectConversation, loadMessages, upsertMessage, confirmMessage,
    setTyping, updateConversationPreview, markRead, invalidateHistories, reset, clearError }
})
