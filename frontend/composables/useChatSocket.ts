import type { ClientEvent, ServerEvent, SocketStatus } from '~/types'
import { useApi } from '~/composables/useApi'

type ServerEventHandler = (event: ServerEvent) => void
let socket: WebSocket | null = null
let reconnectTimer: ReturnType<typeof setTimeout> | null = null
let retryCount = 0
let manuallyClosed = false
let connectionGeneration = 0
const handlers = new Set<ServerEventHandler>()

export function useChatSocket() {
  const config = useRuntimeConfig()
  const status = useState<SocketStatus>('chat-socket-status', () => 'idle')

  function onEvent(handler: ServerEventHandler) {
    handlers.add(handler)
    return () => { handlers.delete(handler) }
  }
  function send(event: ClientEvent) {
    if (!socket || socket.readyState !== WebSocket.OPEN) return false
    socket.send(JSON.stringify(event))
    return true
  }
  async function connect() {
    if (!import.meta.client || status.value === 'open' || status.value === 'connecting') return
    manuallyClosed = false
    const generation = ++connectionGeneration
    const isCurrent = () => generation === connectionGeneration && !manuallyClosed
    status.value = 'connecting'
    try {
      const response = await useApi<{ ticket: string }>('/api/ws/ticket', { method: 'POST' })
      if (!isCurrent()) return
      if (!response.ticket) throw new Error('Ticket vacío')
      const url = new URL('/ws', config.public.wsBase)
      url.searchParams.set('ticket', response.ticket)
      const connection = new WebSocket(url.toString())
      socket = connection
      const ownsConnection = () => isCurrent() && socket === connection
      connection.onopen = () => {
        if (!ownsConnection()) { connection.close(); return }
        retryCount = 0
        status.value = 'open'
      }
      connection.onmessage = event => {
        if (!ownsConnection()) return
        try {
          const parsed = JSON.parse(event.data as string) as ServerEvent
          handlers.forEach(handler => handler(parsed))
        } catch { return }
      }
      connection.onclose = event => {
        if (!ownsConnection()) return
        socket = null
        status.value = 'closed'
        if (event.code === 4001) {
          manuallyClosed = true
          for (const handler of handlers) handler({ type: 'auth.expired' })
          return
        }
        scheduleReconnect()
      }
      connection.onerror = () => {
        if (ownsConnection()) connection.close()
      }
    } catch (error) {
      if (!isCurrent()) return
      status.value = 'closed'
      if ((error as { statusCode?: number }).statusCode === 401) {
        manuallyClosed = true
        for (const handler of handlers) handler({ type: 'auth.expired' })
      }
      scheduleReconnect()
    }
  }
  function scheduleReconnect() {
    if (reconnectTimer || manuallyClosed) return
    const delay = Math.min(1000 * 2 ** retryCount, 15000)
    retryCount++
    reconnectTimer = setTimeout(() => { reconnectTimer = null; void connect() }, delay)
  }
  function disconnect() {
    manuallyClosed = true
    connectionGeneration++
    if (reconnectTimer) { clearTimeout(reconnectTimer); reconnectTimer = null }
    const previous = socket
    socket = null
    previous?.close()
    retryCount = 0
    status.value = 'idle'
  }
  return { status, connect, disconnect, send, onEvent }
}
