import type { ClientEvent, ServerEvent, SocketStatus } from '~/types'
import { useApi } from '~/composables/useApi'

type ServerEventHandler = (event: ServerEvent) => void

let socket: WebSocket | null = null
let reconnectTimer: ReturnType<typeof setTimeout> | null = null
let retryCount = 0
let manuallyClosed = false
const handlers = new Set<ServerEventHandler>()

export function useChatSocket() {
  const config = useRuntimeConfig()
  const status = useState<SocketStatus>('chat-socket-status', () => 'idle')

  function onEvent(handler: ServerEventHandler) {
    handlers.add(handler)
    return () => {
      handlers.delete(handler)
    }
  }

  function send(event: ClientEvent) {
    if (!socket || socket.readyState !== WebSocket.OPEN) {
      return false
    }
    socket.send(JSON.stringify(event))
    return true
  }

  async function connect() {
    if (!import.meta.client || status.value === 'open' || status.value === 'connecting') {
      return
    }

    manuallyClosed = false
    status.value = 'connecting'

    try {
      const response = await useApi<{ ticket: string }>('/api/ws/ticket', {
        method: 'POST',
      })

      if (!response.ticket) {
        throw new Error('Ticket vacío')
      }

      const url = new URL('/ws', config.public.wsBase)
      url.searchParams.set('ticket', response.ticket)

      socket = new WebSocket(url.toString())

      socket.onopen = () => {
        retryCount = 0
        status.value = 'open'
      }

      socket.onmessage = (event) => {
        try {
          const parsed = JSON.parse(event.data as string) as ServerEvent
          handlers.forEach((handler) => handler(parsed))
        } catch {
          return
        }
      }

      socket.onclose = () => {
        status.value = 'closed'
        if (!manuallyClosed) {
          scheduleReconnect()
        }
      }

      socket.onerror = () => {
        socket?.close()
      }
    } catch {
      status.value = 'closed'
      scheduleReconnect()
    }
  }

  function scheduleReconnect() {
    if (reconnectTimer || manuallyClosed) {
      return
    }

    const delay = Math.min(1000 * 2 ** retryCount, 15000)
    retryCount += 1

    reconnectTimer = setTimeout(() => {
      reconnectTimer = null
      void connect()
    }, delay)
  }

  function disconnect() {
    manuallyClosed = true

    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }

    socket?.close()
    socket = null
    status.value = 'idle'
  }

  return {
    status,
    connect,
    disconnect,
    send,
    onEvent,
  }
}
