<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'
import { useChatStore } from '~/stores/chat'
import { canWriteConversation, conversationTitle } from '~/utils/conversations'
import type { Message } from '~/types'
const loadingEarlier = ref(false)
const showPeople = ref(false)
const deletingMessage = ref<Message | null>(null)
const deletingChat = ref(false)
const deleteBusy = ref(false)

const chat = useChatStore()
const auth = useAuthStore()
const isGroup = computed(() => chat.activeConversation?.kind === 'group')
const title = computed(() => chat.activeConversation ? conversationTitle(chat.activeConversation) : '')
watch(() => chat.activeConversationId, () => { showPeople.value = false; deletingMessage.value = null; deletingChat.value = false })
async function confirmDeletion() {
  if (deleteBusy.value) return
  const message = deletingMessage.value
  const id = chat.activeConversationId
  if (!message && !id) return
  deleteBusy.value = true
  try {
    if (message) await chat.deleteMessage(message.conversationId, message.id)
    else await chat.deleteConversation(id!)
    deletingMessage.value = null
    deletingChat.value = false
  } finally { deleteBusy.value = false }
}

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
    class="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden rounded-2xl border border-slate-200 bg-white dark:border-slate-800 dark:bg-slate-900"
  >
    <header class="flex shrink-0 items-center gap-3 border-b border-slate-200 px-5 py-4 dark:border-slate-800">
      <button
        type="button"
        class="rounded-lg border border-slate-300 dark:border-slate-600 px-2 py-1 text-sm text-slate-600 dark:text-slate-300 md:hidden"
        @click="chat.activeConversationId = null"
      >
        Volver
      </button>

      <ChatAvatar :name="title" :image="!isGroup ? chat.activeConversation.peer.avatarUrl : undefined" :group="isGroup" :online="!isGroup && chat.activeConversation.peer.id ? !!chat.onlineUsers[chat.activeConversation.peer.id] : undefined" />

      <div class="min-w-0 flex-1">
        <p class="flex items-center gap-2 font-medium"><span class="truncate">{{ title }}</span><AccountBadge v-if="!isGroup && chat.activeConversation.peer.id" :provider="chat.activeConversation.peer.authProvider" /></p>
        <p v-if="isGroup" class="text-xs text-slate-500 dark:text-slate-400">{{ chat.activeConversation.members?.length || 0 }} participantes</p>
        <p v-else-if="chat.onlineUsers[chat.activeConversation.peer.id]" class="text-xs text-slate-500 dark:text-slate-400">En línea</p>
      </div>
      <button v-if="isGroup" type="button" :aria-expanded="showPeople" class="rounded px-2 py-1.5 text-sm text-slate-600 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-800" @click="showPeople = !showPeople">{{ showPeople ? 'Cerrar' : 'Personas' }}</button>
      <details class="relative"><summary class="flex size-8 cursor-pointer list-none items-center justify-center rounded-lg text-slate-500 hover:bg-slate-100 dark:hover:bg-slate-800 [&::-webkit-details-marker]:hidden" aria-label="Opciones del chat"><ChatIcon name="more" :size="18" /></summary><div class="absolute right-0 top-full z-30 mt-2 w-44 rounded-xl border border-slate-200 bg-white p-1.5 shadow-lg dark:border-slate-700 dark:bg-slate-900"><button type="button" class="flex w-full items-center gap-2 rounded-lg px-3 py-2.5 text-sm text-red-600 hover:bg-red-50 dark:hover:bg-red-950" @click="deletingChat = true; ($event.currentTarget as HTMLElement).closest('details')?.removeAttribute('open')"><ChatIcon name="trash" :size="17" />Eliminar chat</button></div></details>
    </header>
    <GroupDetails v-if="isGroup && showPeople" :key="chat.activeConversation.id" :conversation="chat.activeConversation" />

    <div ref="scroller" class="chat-scroll min-h-0 flex-1 space-y-6 overflow-y-auto overscroll-contain px-5 py-6 sm:px-6">
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
        :avatar="isGroup ? chat.activeConversation.members?.find(member => member.id === message.senderId)?.avatarUrl : chat.activeConversation.peer.avatarUrl"
        :deletable="true"
        @delete="deletingMessage = $event"
      />
    </div>

    <TypingIndicator :active="peerTyping" />

    <p v-if="!canWriteConversation(chat.activeConversation)" class="shrink-0 border-t border-slate-200 p-3 text-center text-sm text-slate-500 dark:border-slate-700 dark:text-slate-400">Esta cuenta de invitado ya no está disponible. Aquí se conservan los mensajes que enviaste.</p>
    <MessageComposer v-else :key="chat.activeConversation.id" :conversation-id="chat.activeConversation.id" />
    <ConfirmDialog :open="!!deletingMessage || deletingChat" :title="deletingMessage ? '¿Eliminar este mensaje?' : '¿Eliminar este chat?'" :message="deletingMessage ? 'El mensaje se eliminará para todos los participantes. Esta acción no se puede deshacer.' : 'Se quitará de tu lista y se limpiará tu historial visible. Los demás conservarán sus mensajes. Volverá a aparecer si recibes nuevos mensajes.'" :busy="deleteBusy" @cancel="deletingMessage = null; deletingChat = false" @confirm="confirmDeletion" />
  </section>

  <section v-else class="flex min-w-0 flex-1 flex-col items-center justify-center rounded-2xl border border-slate-200 bg-white p-8 text-center dark:border-slate-800 dark:bg-slate-900 max-md:hidden">
    <span class="mb-4 flex size-16 items-center justify-center rounded-2xl bg-[#eef2ff] text-[#465fff] dark:bg-slate-800"><ChatIcon name="chat" :size="30" /></span>
    <h2 class="text-lg font-semibold">Tus conversaciones</h2>
    <p class="mt-2 max-w-xs text-sm leading-6 text-slate-400">Selecciona un chat o inicia una conversación desde el botón de nuevo mensaje.</p>
  </section>
</template>
