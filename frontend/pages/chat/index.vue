<script setup lang="ts">
import { watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { useChatStore } from '~/stores/chat'
import { useAuthStore } from '~/stores/auth'
import { useChatSocket } from '~/composables/useChatSocket'
const chat = useChatStore()
const auth = useAuthStore()
const socket = useChatSocket()
let stopListening: (() => void) | undefined
const readSent = new Map<string, string>()
function markVisibleRead() {
  if (document.visibilityState !== 'visible' || !document.hasFocus() || socket.status.value !== 'open') return
  const id = chat.activeConversationId
  const message = chat.activeMessages.filter(item => item.status !== 'sending').at(-1)
  if (id && message && readSent.get(id) !== message.id && socket.send({ type: 'message.read', conversationId: id, messageId: message.id })) readSent.set(id, message.id)
}
onMounted(async () => {
  stopListening = socket.onEvent(event => {
    switch (event.type) {
      case 'message.new': {
        const inserted = chat.upsertMessage({ ...event.message, status: 'sent' })
        const conversation = chat.conversations.find(item => item.id === event.message.conversationId)
        if (inserted && conversation && event.message.senderId !== auth.user?.id) conversation.unreadCount++
        chat.updateConversationPreview(event.message)
        break
      }
      case 'message.ack':
        chat.confirmMessage(event.clientMessageId, event.message)
        chat.updateConversationPreview(event.message)
        break
      case 'message.read':
        chat.markRead(event.conversationId, event.userId, event.messageId)
        if (event.userId === auth.user?.id) void chat.loadConversations()
        break
      case 'typing': chat.setTyping(event.conversationId, event.userId, event.active); break
      case 'presence':
        chat.onlineUsers[event.userId] = event.online
        if (!event.online) for (const id of Object.keys(chat.typingByConversation)) chat.setTyping(id, event.userId, false)
        break
      case 'error': chat.errorMessage = event.error; break
    }
  })
  document.addEventListener('visibilitychange', markVisibleRead)
  window.addEventListener('focus', markVisibleRead)
  await chat.loadConversations()
  await socket.connect()
})
onBeforeUnmount(() => {
  stopListening?.()
  document.removeEventListener('visibilitychange', markVisibleRead)
  window.removeEventListener('focus', markVisibleRead)
  socket.disconnect()
})
watch(() => chat.activeConversationId, (id, previous) => {
  if (socket.status.value !== 'open') return
  if (previous) { socket.send({ type: 'typing', conversationId: previous, active: false }); socket.send({ type: 'conversation.leave', conversationId: previous }) }
  if (id) socket.send({ type: 'conversation.join', conversationId: id })
})
watch(() => socket.status.value, status => {
  if (status === 'open') {
    readSent.clear()
    // Refresh history after reconnecting so messages sent while offline are recovered.
    void chat.loadConversations()
    if (chat.activeConversationId) {
      socket.send({ type: 'conversation.join', conversationId: chat.activeConversationId })
      void chat.loadMessages(chat.activeConversationId)
    }
  } else chat.onlineUsers = {}
})
watch([() => chat.activeConversationId, () => chat.activeMessages.at(-1)?.id, () => socket.status.value], () => {
  if (import.meta.client) void nextTick(markVisibleRead)
})
</script>
<template>
  <div class="flex flex-1 flex-col">
    <div
      v-if="chat.errorMessage"
      class="mb-3 flex items-center justify-between rounded-lg bg-amber-50 dark:bg-amber-950 px-3 py-2 text-sm text-amber-800 dark:text-amber-200"
    >
      <span>{{ chat.errorMessage }}</span>
      <button type="button" class="font-medium" @click="chat.clearError()">Cerrar</button>
    </div>

    <div
      class="flex min-h-[32rem] flex-1 overflow-hidden rounded-2xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 shadow-sm"
    >
      <ChatList />
      <ChatWindow />
    </div>
  </div>
</template>
