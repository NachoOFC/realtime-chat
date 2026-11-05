import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Message } from '~/types'
import { useApi } from '~/composables/useApi'
import {
  ASSISTANT_CONVERSATION_ID,
  ASSISTANT_SENDER_ID,
  MAX_ASSISTANT_USER_RUNES,
  buildAssistantHistory,
  runeLength,
} from '~/utils/assistant'

interface AssistantStatus {
  enabled: boolean
  provider: 'gemini'
  model: string
}

function requestError(error: unknown): string {
  const value = error as { statusCode?: number; status?: number } | null
  const status = value?.statusCode ?? value?.status
  if (status === 401) return 'Tu sesión terminó. Vuelve a iniciar sesión para continuar.'
  if (status === 429) return 'IACHAT alcanzó su límite de uso. Intenta más tarde.'
  return 'IACHAT no pudo responder. Intenta de nuevo en un momento.'
}

export const useAssistantStore = defineStore('assistant', () => {
  const messages = ref<Message[]>([])
  const sending = ref(false)
  const enabled = ref(false)
  const loadingStatus = ref(false)
  const statusLoaded = ref(false)
  const errorMessage = ref<string | null>(null)
  let sessionEpoch = 0
  let statusController: AbortController | undefined
  let sendController: AbortController | undefined
  let statusTimeout: ReturnType<typeof setTimeout> | undefined
  let sendTimeout: ReturnType<typeof setTimeout> | undefined

  async function loadStatus() {
    if (loadingStatus.value) return
    const epoch = sessionEpoch
    const controller = new AbortController()
    statusController = controller
    loadingStatus.value = true
    errorMessage.value = null
    statusTimeout = setTimeout(() => controller.abort(), 20000)
    try {
      const status = await useApi<AssistantStatus>('/api/assistant', { signal: controller.signal })
      if (epoch !== sessionEpoch) return
      if (controller.signal.aborted) throw new Error('request aborted')
      enabled.value = status.enabled === true
    } catch {
      if (epoch !== sessionEpoch) return
      enabled.value = false
      errorMessage.value = 'No pudimos comprobar si IACHAT está disponible. Actualiza su estado e intenta de nuevo.'
    } finally {
      if (epoch === sessionEpoch && statusController === controller) {
        clearTimeout(statusTimeout)
        statusTimeout = undefined
        statusController = undefined
        statusLoaded.value = true
        loadingStatus.value = false
      }
    }
  }

  async function send(content: string, senderId: string): Promise<boolean> {
    if (sending.value) return false
    if (!enabled.value || loadingStatus.value) {
      errorMessage.value = 'IACHAT todavía no está disponible. Actualiza su estado e intenta de nuevo.'
      return false
    }
    const text = content.trim()
    if (!senderId) {
      errorMessage.value = 'Inicia sesión para conversar con IACHAT.'
      return false
    }
    if (!text || runeLength(text) > MAX_ASSISTANT_USER_RUNES) {
      errorMessage.value = 'Escribe un mensaje de hasta 4000 caracteres.'
      return false
    }

    const epoch = sessionEpoch
    const history = buildAssistantHistory(messages.value, senderId, text)
    const controller = new AbortController()
    sendController = controller
    const pendingId = crypto.randomUUID()
    messages.value.push({
      id: pendingId,
      clientMessageId: pendingId,
      conversationId: ASSISTANT_CONVERSATION_ID,
      senderId,
      content: text,
      createdAt: new Date().toISOString(),
      status: 'sending',
    })
    sending.value = true
    errorMessage.value = null
    let timedOut = false
    sendTimeout = setTimeout(() => { timedOut = true; controller.abort() }, 45000)

    try {
      const result = await useApi<{ content: string }>('/api/assistant/messages', {
        method: 'POST',
        body: { messages: history },
        signal: controller.signal,
      })
      if (epoch !== sessionEpoch) return false
      if (controller.signal.aborted) throw new Error('request aborted')
      const answer = typeof result.content === 'string' ? result.content.trim() : ''
      if (!answer) throw new Error('empty assistant response')
      messages.value = [
        ...messages.value.map(message => message.id === pendingId ? { ...message, status: 'sent' as const } : message),
        {
          id: crypto.randomUUID(),
          conversationId: ASSISTANT_CONVERSATION_ID,
          senderId: ASSISTANT_SENDER_ID,
          content: answer,
          createdAt: new Date().toISOString(),
          status: 'sent',
        },
      ]
      return true
    } catch (error) {
      if (epoch !== sessionEpoch) return false
      messages.value = messages.value.filter(message => message.id !== pendingId)
      errorMessage.value = timedOut ? 'La respuesta tardó demasiado. Intenta de nuevo.' : requestError(error)
      return false
    } finally {
      if (epoch === sessionEpoch && sendController === controller) {
        clearTimeout(sendTimeout)
        sendTimeout = undefined
        sendController = undefined
        sending.value = false
      }
    }
  }

  function reset() {
    sessionEpoch++
    statusController?.abort()
    sendController?.abort()
    clearTimeout(statusTimeout)
    clearTimeout(sendTimeout)
    statusController = undefined
    sendController = undefined
    statusTimeout = undefined
    sendTimeout = undefined
    messages.value = []
    sending.value = false
    enabled.value = false
    loadingStatus.value = false
    statusLoaded.value = false
    errorMessage.value = null
  }

  return { messages, sending, enabled, loadingStatus, statusLoaded, errorMessage, loadStatus, send, reset }
})
