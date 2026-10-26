import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import type { Conversation, Message, MessagePage } from '~/types'
import { useApi } from '~/composables/useApi'

export const useChatStore = defineStore('chat', () => {
  const conversations = ref<Conversation[]>([])
  const activeConversationId = ref<string | null>(null)
  const messages = ref<Record<string, Message[]>>({})
  const typingByConversation = ref<Record<string, string[]>>({})
  const loadingConversations = ref(false)
  const errorMessage = ref<string | null>(null)

  const activeConversation = computed(
    () => conversations.value.find((item) => item.id === activeConversationId.value) ?? null,
  )

  const activeMessages = computed(() => {
    if (!activeConversationId.value) {
      return []
    }
    return messages.value[activeConversationId.value] ?? []
  })

  const activeTypingUsers = computed(() => {
    if (!activeConversationId.value) {
      return []
    }
    return typingByConversation.value[activeConversationId.value] ?? []
  })

  async function loadConversations() {
    loadingConversations.value = true
    errorMessage.value = null

    try {
      conversations.value = await useApi<Conversation[]>('/api/conversations')
    } catch {
      errorMessage.value = 'No se pudieron cargar las conversaciones.'
    } finally {
      loadingConversations.value = false
    }
  }

  async function selectConversation(id: string) {
    activeConversationId.value = id

    if (!messages.value[id]) {
      await loadMessages(id)
    }
  }

  async function loadMessages(id: string, before?: string) {
    try {
      const page = await useApi<MessagePage>(`/api/conversations/${id}/messages`, {
        query: { before },
      })

      messages.value[id] = before
        ? [...page.messages, ...(messages.value[id] ?? [])]
        : page.messages
    } catch {
      errorMessage.value = 'No se pudo cargar el historial.'
    }
  }

  function upsertMessage(message: Message) {
    const current = messages.value[message.conversationId] ?? []
    const exists = current.some(
      (item) =>
        item.id === message.id ||
        (message.clientMessageId && item.clientMessageId === message.clientMessageId),
    )

    if (exists) {
      return
    }

    messages.value[message.conversationId] = [...current, message]
  }

  function confirmMessage(clientMessageId: string, message: Message) {
    const current = messages.value[message.conversationId] ?? []
    const index = current.findIndex((item) => item.clientMessageId === clientMessageId)

    if (index === -1) {
      upsertMessage({ ...message, status: 'sent' })
      return
    }

    messages.value[message.conversationId] = current.map((item, itemIndex) =>
      itemIndex === index ? { ...message, status: 'sent' } : item,
    )
  }

  function setTyping(conversationId: string, userId: string, active: boolean) {
    const current = typingByConversation.value[conversationId] ?? []
    typingByConversation.value[conversationId] = active
      ? Array.from(new Set([...current, userId]))
      : current.filter((id) => id !== userId)
  }

  function clearError() {
    errorMessage.value = null
  }

  return {
    conversations,
    activeConversationId,
    messages,
    typingByConversation,
    loadingConversations,
    errorMessage,
    activeConversation,
    activeMessages,
    activeTypingUsers,
    loadConversations,
    selectConversation,
    loadMessages,
    upsertMessage,
    confirmMessage,
    setTyping,
    clearError,
  }
})
