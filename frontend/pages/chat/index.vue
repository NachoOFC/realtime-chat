<script setup lang="ts">
import { onBeforeUnmount, onMounted, watch } from 'vue'
import { useChatStore } from '~/stores/chat'
import { useChatSocket } from '~/composables/useChatSocket'

const chat = useChatStore()
const socket = useChatSocket()

let stopListening: (() => void) | undefined

onMounted(async () => {
  stopListening = socket.onEvent((event) => {
    switch (event.type) {
      case 'message.new':
        chat.upsertMessage({ ...event.message, status: 'sent' })
        break
      case 'message.ack':
        chat.confirmMessage(event.clientMessageId, event.message)
        break
      case 'typing':
        chat.setTyping(event.conversationId, event.userId, event.active)
        break
      case 'error':
        chat.errorMessage = event.error
        break
    }
  })

  await chat.loadConversations()
  await socket.connect()
})

onBeforeUnmount(() => {
  stopListening?.()
  socket.disconnect()
})

watch(
  () => chat.activeConversationId,
  (conversationId, previousConversationId) => {
    if (socket.status.value !== 'open') {
      return
    }

    if (previousConversationId) {
      socket.send({ type: 'conversation.leave', conversationId: previousConversationId })
    }

    if (conversationId) {
      socket.send({ type: 'conversation.join', conversationId })
    }
  },
)

watch(
  () => socket.status.value,
  (status) => {
    if (status === 'open' && chat.activeConversationId) {
      socket.send({ type: 'conversation.join', conversationId: chat.activeConversationId })
    }
  },
)
</script>

<template>
  <div class="flex flex-1 flex-col">
    <div
      v-if="chat.errorMessage"
      class="mb-3 flex items-center justify-between rounded-lg bg-amber-50 px-3 py-2 text-sm text-amber-800"
    >
      <span>{{ chat.errorMessage }}</span>
      <button type="button" class="font-medium" @click="chat.clearError()">Cerrar</button>
    </div>

    <div
      class="flex min-h-[32rem] flex-1 overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm"
    >
      <ChatList />
      <ChatWindow />
    </div>
  </div>
</template>
