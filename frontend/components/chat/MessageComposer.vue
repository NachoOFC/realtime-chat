<script setup lang="ts">
import type { Message } from '~/types'
import { useAuthStore } from '~/stores/auth'
import { useChatStore } from '~/stores/chat'
import { useChatSocket } from '~/composables/useChatSocket'
import { canWriteConversation } from '~/utils/conversations'

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
  if (canWriteConversation(chat.conversations.find(item => item.id === props.conversationId))) socket.send({ type: 'typing', conversationId: props.conversationId, active: false })
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
  <form class="flex shrink-0 items-center gap-3 border-t border-slate-200 px-5 py-4 dark:border-slate-800" @submit.prevent="submit">
    <input
      v-model="content"
      @input="announceTyping"
      @blur="stopTyping"
      type="text"
      maxlength="4000"
      autocomplete="off"
      placeholder="Escribe un mensaje…"
      aria-label="Mensaje"
      class="min-w-0 flex-1 rounded-xl border border-slate-200 bg-white px-4 py-3 text-sm outline-none transition placeholder:text-slate-400 focus:border-[#465fff] focus:ring-2 focus:ring-[#465fff]/10 dark:border-slate-700 dark:bg-slate-900"
    >
    <button
      type="submit"
      :disabled="!content.trim()"
      aria-label="Enviar mensaje"
      class="flex size-11 shrink-0 items-center justify-center rounded-xl bg-[#465fff] text-white transition hover:bg-[#3641f5] disabled:bg-[#c7d2fe] disabled:dark:bg-slate-700"
    >
      <ChatIcon name="send" :size="19" />
    </button>
  </form>
</template>
