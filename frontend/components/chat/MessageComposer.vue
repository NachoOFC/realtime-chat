<script setup lang="ts">
import type { Message } from '~/types'
import { useAuthStore } from '~/stores/auth'
import { useChatStore } from '~/stores/chat'
import { useChatSocket } from '~/composables/useChatSocket'
import { useDemoMode } from '~/composables/useDemoMode'

const props = defineProps<{
  conversationId: string
}>()

const auth = useAuthStore()
const chat = useChatStore()
const socket = useChatSocket()
const demoMode = useDemoMode()

const content = ref('')

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
    status: demoMode.value ? 'sent' : 'sending',
  }

  chat.upsertMessage(optimistic)
  content.value = ''

  if (demoMode.value) {
    chat.updateConversationPreview(optimistic)
    chat.simulateBotReply(props.conversationId, text)
    return
  }

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
