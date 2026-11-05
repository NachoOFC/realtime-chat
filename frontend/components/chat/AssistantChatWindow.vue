<script setup lang="ts">
import MessageBubble from '~/components/chat/MessageBubble.vue'
import { useAssistantStore } from '~/stores/assistant'
import { useAuthStore } from '~/stores/auth'
import { useChatStore } from '~/stores/chat'

const assistant = useAssistantStore()
const auth = useAuthStore()
const chat = useChatStore()
const content = ref('')
const scroller = ref<HTMLElement | null>(null)
const canWrite = computed(() => assistant.enabled && !assistant.loadingStatus && !assistant.sending && !!auth.user?.id)

async function scrollToBottom() {
  await nextTick()
  if (scroller.value) scroller.value.scrollTop = scroller.value.scrollHeight
}

onMounted(() => {
  if (!assistant.statusLoaded) void assistant.loadStatus()
  void scrollToBottom()
})
watch(() => assistant.messages.length, scrollToBottom)
watch(() => assistant.sending, scrollToBottom)
watch(() => auth.user?.id, () => { content.value = '' })

async function submit() {
  const text = content.value.trim()
  const senderId = auth.user?.id
  if (!text || !senderId || !canWrite.value) return
  content.value = ''
  const sent = await assistant.send(text, senderId)
  if (!sent && auth.user?.id === senderId && !content.value) content.value = text
}
</script>

<template>
  <section class="flex min-w-0 flex-1 flex-col" aria-label="Chat con IACHAT">
    <header class="flex items-center gap-3 border-b border-slate-200 px-4 py-3 dark:border-slate-700">
      <button type="button" class="rounded-lg border border-slate-300 px-2 py-1 text-sm text-slate-600 dark:border-slate-600 dark:text-slate-300 md:hidden" @click="chat.activeConversationId = null">
        Volver
      </button>
      <span class="flex size-9 shrink-0 items-center justify-center rounded-full bg-sky-100 text-sky-700 dark:bg-sky-950 dark:text-sky-300" aria-hidden="true">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" class="size-5">
          <path stroke-linecap="round" stroke-linejoin="round" d="m12 3 2.5 6.5L21 12l-6.5 2.5L12 21l-2.5-6.5L3 12l6.5-2.5L12 3Z" />
        </svg>
      </span>
      <div>
        <p class="flex items-center gap-2 font-medium">IACHAT <span class="rounded-full bg-sky-100 px-2 py-0.5 text-[10px] font-semibold text-sky-800 dark:bg-sky-950 dark:text-sky-300">IA</span></p>
        <p class="text-xs text-slate-500 dark:text-slate-400">Tu asistente de inteligencia artificial</p>
      </div>
    </header>

    <div ref="scroller" class="flex-1 space-y-3 overflow-y-auto bg-white px-4 py-4 dark:bg-slate-900" role="log" aria-label="Mensajes con IACHAT" aria-live="polite">
      <div v-if="!assistant.messages.length" class="mx-auto max-w-sm px-3 py-12 text-center">
        <h2 class="text-lg font-semibold text-slate-900 dark:text-slate-100">Hola, soy IACHAT</h2>
        <p class="mt-2 text-sm leading-6 text-slate-500 dark:text-slate-400">Pregunta, explora ideas o pide ayuda. Esta conversación es solo entre tú y el asistente.</p>
      </div>

      <MessageBubble v-for="message in assistant.messages" :key="message.id" :message="message" :own="message.senderId === auth.user?.id" />

      <p v-if="assistant.sending" class="flex items-center gap-2 py-2 text-sm text-slate-500 dark:text-slate-400" role="status">
        <span class="size-2 animate-pulse rounded-full bg-sky-500" aria-hidden="true" />
        IACHAT está pensando…
      </p>
    </div>

    <div v-if="assistant.loadingStatus" class="border-t border-slate-200 px-4 py-3 text-sm text-slate-500 dark:border-slate-700 dark:text-slate-400" role="status">
      Consultando disponibilidad de IACHAT…
    </div>
    <div v-else-if="!assistant.enabled || assistant.errorMessage" class="border-t border-slate-200 px-4 py-3 text-sm dark:border-slate-700" :role="assistant.errorMessage ? 'alert' : 'status'">
      <p class="text-slate-600 dark:text-slate-300">{{ assistant.errorMessage || 'IACHAT aún no está disponible. Puedes seguir conversando con otras personas mientras tanto.' }}</p>
      <button v-if="!assistant.enabled" type="button" class="mt-2 rounded-lg border border-slate-300 px-3 py-1.5 text-xs font-medium hover:bg-slate-100 dark:border-slate-600 dark:hover:bg-slate-800" @click="assistant.loadStatus()">
        Actualizar estado
      </button>
    </div>

    <footer class="border-t border-slate-200 dark:border-slate-700">
      <p class="px-4 pt-3 text-[11px] leading-4 text-slate-500 dark:text-slate-400">Los mensajes de este chat se envían a Google Gemini. El historial se borra al recargar o salir.</p>
      <form class="flex items-end gap-2 p-3" @submit.prevent="submit">
        <textarea
          v-model="content"
          rows="1"
          maxlength="4000"
          :disabled="!canWrite"
          aria-label="Mensaje para IACHAT"
          placeholder="Escribe a IACHAT"
          class="max-h-32 min-h-10 min-w-0 flex-1 resize-y rounded-2xl border border-slate-300 bg-white px-4 py-2 text-sm outline-none focus:border-sky-500 disabled:cursor-not-allowed disabled:opacity-60 dark:border-slate-600 dark:bg-slate-900 dark:focus:border-sky-400"
          @keydown.enter.exact.prevent="submit"
        />
        <button type="submit" :disabled="!canWrite || !content.trim()" class="min-h-10 rounded-full bg-slate-900 px-5 py-2 text-sm font-medium text-white transition hover:bg-slate-800 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-sky-600 dark:hover:bg-sky-500">
          {{ assistant.sending ? 'Esperando…' : 'Enviar' }}
        </button>
      </form>
    </footer>
  </section>
</template>
