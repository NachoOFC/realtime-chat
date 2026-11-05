<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'
import { useChatStore } from '~/stores/chat'
import { canWriteConversation, conversationTitle } from '~/utils/conversations'
const loadingEarlier = ref(false)
const showPeople = ref(false)

const chat = useChatStore()
const auth = useAuthStore()
const isGroup = computed(() => chat.activeConversation?.kind === 'group')
const title = computed(() => chat.activeConversation ? conversationTitle(chat.activeConversation) : '')
watch(() => chat.activeConversationId, () => { showPeople.value = false })

const scroller = ref<HTMLElement | null>(null)

const peerTyping = computed(() =>
  chat.activeTypingUsers.some((userId: string) => userId !== auth.user?.id),
)

async function scrollToBottom() {
  await nextTick()
  const element = scroller.value
  if (element) {
    element.scrollTop = element.scrollHeight
  }
}

watch(() => chat.activeConversationId, scrollToBottom)
watch(() => chat.activeMessages.length, () => { if (!loadingEarlier.value) void scrollToBottom() })
async function loadEarlier() {
  const id = chat.activeConversationId
  const cursor = id ? chat.nextCursors[id] : undefined
  if (!id || !cursor || loadingEarlier.value) return
  loadingEarlier.value = true
  const element = scroller.value
  const previousHeight = element?.scrollHeight ?? 0
  try {
    await chat.loadMessages(id, cursor)
    await nextTick()
    if (element) element.scrollTop += element.scrollHeight - previousHeight
  } finally { loadingEarlier.value = false }
}
</script>

<template>
  <section
    v-if="chat.activeConversation"
    class="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden"
  >
    <header class="flex shrink-0 items-center gap-3 border-b border-slate-200 dark:border-slate-700 px-4 py-3">
      <button
        type="button"
        class="rounded-lg border border-slate-300 dark:border-slate-600 px-2 py-1 text-sm text-slate-600 dark:text-slate-300 md:hidden"
        @click="chat.activeConversationId = null"
      >
        Volver
      </button>

      <span
        class="flex size-9 items-center justify-center rounded-full bg-slate-200 dark:bg-slate-700 text-sm font-semibold text-slate-700 dark:text-slate-200"
      >
        {{ title.slice(0, 1).toUpperCase() }}
      </span>

      <div class="min-w-0 flex-1">
        <p class="flex items-center gap-2 font-medium"><span class="truncate">{{ title }}</span><AccountBadge v-if="!isGroup && chat.activeConversation.peer.id" :provider="chat.activeConversation.peer.authProvider" /></p>
        <p v-if="isGroup" class="text-xs text-slate-500 dark:text-slate-400">{{ chat.activeConversation.members?.length || 0 }} participantes</p>
        <p v-else-if="chat.onlineUsers[chat.activeConversation.peer.id]" class="text-xs text-slate-500 dark:text-slate-400">En línea</p>
      </div>
      <button v-if="isGroup" type="button" :aria-expanded="showPeople" class="rounded px-2 py-1.5 text-sm text-slate-600 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-800" @click="showPeople = !showPeople">{{ showPeople ? 'Cerrar' : 'Personas' }}</button>
    </header>
    <GroupDetails v-if="isGroup && showPeople" :key="chat.activeConversation.id" :conversation="chat.activeConversation" />

    <div ref="scroller" class="min-h-0 flex-1 space-y-2 overflow-y-auto overscroll-contain bg-white dark:bg-slate-900 px-4 py-4">
      <button v-if="chat.nextCursors[chat.activeConversation.id]" type="button" :disabled="loadingEarlier" class="mx-auto block rounded-lg border border-slate-300 px-3 py-1 text-xs dark:border-slate-600" @click="loadEarlier">{{ loadingEarlier ? 'Cargando...' : 'Ver mensajes anteriores' }}</button>
      <p v-if="!chat.activeMessages.length" class="py-10 text-center text-sm text-slate-500 dark:text-slate-400">
        Todavía no hay mensajes en esta conversación.
      </p>

      <MessageBubble
        v-for="message in chat.activeMessages"
        :key="message.id"
        :message="message"
        :own="message.senderId === auth.user?.id"
        :sender="isGroup ? message.senderName || chat.activeConversation.members?.find(member => member.id === message.senderId)?.username || 'Participante' : undefined"
        :read-label="isGroup ? 'Visto por alguien' : 'Leído'"
      />
    </div>

    <TypingIndicator :active="peerTyping" />

    <p v-if="!canWriteConversation(chat.activeConversation)" class="shrink-0 border-t border-slate-200 p-3 text-center text-sm text-slate-500 dark:border-slate-700 dark:text-slate-400">Esta cuenta de invitado ya no está disponible. Aquí se conservan los mensajes que enviaste.</p>
    <MessageComposer v-else :key="chat.activeConversation.id" :conversation-id="chat.activeConversation.id" />
  </section>

  <section v-else class="flex flex-1 items-center justify-center bg-white dark:bg-slate-900 max-md:hidden">
    <p class="text-sm text-slate-500 dark:text-slate-400">Selecciona una conversación para empezar.</p>
  </section>
</template>
