<script setup lang="ts">
import type { Message } from '~/types'
import { useAuthStore } from '~/stores/auth'
import { useChatStore } from '~/stores/chat'
import { useChatSocket } from '~/composables/useChatSocket'

const props = defineProps<{
  conversationId: string
}>()

const auth = useAuthStore()
const chat = useChatStore()
const socket = useChatSocket()

const content = ref('')
let typingTimer: ReturnType<typeof setTimeout> | undefined
function stopTyping() {
  clearTimeout(typingTimer)
  if (chat.conversations.find(item => item.id === props.conversationId)?.peer.id) socket.send({ type: 'typing', conversationId: props.conversationId, active: false })
}
function announceTyping() {
  socket.send({ type: 'typing', conversationId: props.conversationId, active: !!content.value.trim() })
  clearTimeout(typingTimer)
  typingTimer = setTimeout(stopTyping, 1200)
}
onBeforeUnmount(stopTyping)
watch(() => props.conversationId, () => { clearTimeout(typingTimer); content.value = '' })

function submit() {
  const text = content.value.trim()
  const senderId = auth.user?.id

  if (!text || !senderId) {
    return
  }

  const clientMessageId = crypto.randomUUID()
  const optimistic: Message = {
    id: clientMessageId,
    conversationId: props.conversationId,
    senderId,
    content: text,
    clientMessageId,
    createdAt: new Date().toISOString(),
    status: 'sending',
  }

  stopTyping()
  chat.upsertMessage(optimistic)
  content.value = ''

  const sent = socket.send({
    type: 'message.send',
    conversationId: props.conversationId,
    clientMessageId,
    content: text,
  })

  if (!sent) {
    chat.errorMessage = 'Sin conexión en tiempo real. El mensaje quedó pendiente.'
  }
}
</script>

<template>
  <form class="flex gap-2 border-t border-slate-200 dark:border-slate-700 p-3" @submit.prevent="submit">
    <input
      v-model="content"
      @input="announceTyping"
      @blur="stopTyping"
      type="text"
      maxlength="4000"
      autocomplete="off"
      placeholder="Escribe un mensaje"
      class="min-w-0 flex-1 rounded-full border border-slate-300 dark:border-slate-600 px-4 py-2 text-sm bg-white dark:bg-slate-900 outline-none focus:border-slate-500 dark:focus:border-sky-400"
    >
    <button
      type="submit"
      class="rounded-full bg-slate-900 dark:bg-sky-600 px-5 py-2 text-sm font-medium text-white transition hover:bg-slate-800 dark:hover:bg-sky-500"
    >
      Enviar
    </button>
  </form>
</template>
