<script setup lang="ts">
import { useChatStore } from '~/stores/chat'
import { useAssistantStore } from '~/stores/assistant'
import { ASSISTANT_CONVERSATION_ID } from '~/utils/assistant'
const chat = useChatStore()
const assistant = useAssistantStore()
const showUsers = ref(false)
const search = ref('')
const starting = ref(false)
const onlyOnline = ref(false)
const filteredUsers = computed(() => chat.users.filter(user => (!onlyOnline.value || chat.onlineUsers[user.id]) && user.username.toLocaleLowerCase().includes(search.value.trim().toLocaleLowerCase())).sort((a, b) => Number(!!chat.onlineUsers[b.id]) - Number(!!chat.onlineUsers[a.id]) || a.username.localeCompare(b.username)))
async function openUsers() { showUsers.value = true; await chat.loadUsers() }
async function start(id: string) {
  if (starting.value) return
  starting.value = true
  try { if (await chat.startConversation(id)) showUsers.value = false }
  finally { starting.value = false }
}
</script>

<template>
  <aside class="flex w-80 shrink-0 flex-col border-r border-slate-200 bg-slate-50/60 dark:border-slate-700 dark:bg-slate-950/60 max-md:w-full" :class="{ 'max-md:hidden': chat.activeConversationId }">
    <div class="flex items-center justify-between gap-2 border-b border-slate-200 px-4 py-3 dark:border-slate-700">
      <h2 class="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">{{ showUsers ? 'Nuevo chat' : 'Conversaciones' }}</h2>
      <button type="button" class="rounded-lg px-2 py-1 text-sm font-medium text-sky-700 hover:bg-slate-200 dark:text-sky-300 dark:hover:bg-slate-800" @click="showUsers ? showUsers = false : openUsers()">{{ showUsers ? 'Volver' : 'Nuevo chat' }}</button>
    </div>
    <div v-if="showUsers" class="flex flex-col gap-2 p-3">
      <label for="user-search" class="text-sm">Buscar por apodo</label>
      <input id="user-search" v-model="search" type="search" placeholder="Escribe un apodo" class="rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm dark:border-slate-600 dark:bg-slate-900">
      <button type="button" class="self-start text-xs text-sky-700 dark:text-sky-300" :disabled="chat.loadingUsers" @click="chat.loadUsers()">Actualizar usuarios</button>
      <label class="flex items-center gap-2 text-xs"><input v-model="onlyOnline" type="checkbox">Solo conectados</label>
      <p v-if="chat.loadingUsers" class="text-sm text-slate-500 dark:text-slate-400">Cargando usuarios...</p>
      <ul v-else>
        <li v-for="user in filteredUsers" :key="user.id">
          <button type="button" :disabled="starting" class="flex w-full items-center gap-2 rounded-lg px-3 py-3 text-left hover:bg-white disabled:opacity-50 dark:hover:bg-slate-800" @click="start(user.id)"><PresenceDot :online="!!chat.onlineUsers[user.id]" /><span class="min-w-0 flex-1 truncate">{{ user.username }}</span><AccountBadge :provider="user.authProvider" /></button>
        </li>
      </ul>
      <p v-if="!chat.loadingUsers && !filteredUsers.length" class="text-sm text-slate-500 dark:text-slate-400">No hay otros usuarios con ese apodo. Puedes entrar con otro apodo desde una ventana de incógnito.</p>
    </div>
    <template v-else>
      <button type="button" class="flex w-full shrink-0 items-center gap-3 border-b border-slate-200 px-4 py-4 text-left transition hover:bg-white dark:border-slate-700 dark:hover:bg-slate-800" :class="{ 'bg-white dark:bg-slate-900': chat.activeConversationId === ASSISTANT_CONVERSATION_ID }" :aria-pressed="chat.activeConversationId === ASSISTANT_CONVERSATION_ID" @click="chat.selectConversation(ASSISTANT_CONVERSATION_ID)">
        <span class="flex size-10 shrink-0 items-center justify-center rounded-full bg-sky-100 text-sky-700 dark:bg-sky-950 dark:text-sky-300" aria-hidden="true">
          <svg class="size-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><path d="m12 3 2.8 6.2L21 12l-6.2 2.8L12 21l-2.8-6.2L3 12l6.2-2.8L12 3Z" /></svg>
        </span>
        <span class="min-w-0 flex-1">
          <span class="flex items-center justify-between gap-2"><span class="font-medium">IACHAT</span><span class="rounded-full bg-sky-100 px-2 py-0.5 text-[10px] font-medium text-sky-800 dark:bg-sky-950 dark:text-sky-200">IA</span></span>
          <span class="mt-0.5 block truncate text-xs text-slate-500 dark:text-slate-400">{{ assistant.sending ? 'Preparando respuesta…' : assistant.messages.at(-1)?.content || 'Tu asistente con Gemini' }}</span>
        </span>
      </button>
      <p v-if="chat.loadingConversations" class="p-4 text-sm text-slate-500 dark:text-slate-400">Cargando...</p>
      <ul v-else-if="chat.conversations.length" class="flex-1 overflow-y-auto">
        <li v-for="conversation in chat.conversations" :key="conversation.id">
          <button type="button" class="flex w-full items-center gap-3 px-4 py-3 text-left transition hover:bg-white dark:hover:bg-slate-800" :class="{ 'bg-white dark:bg-slate-900': chat.activeConversationId === conversation.id }" @click="chat.selectConversation(conversation.id)">
            <span class="flex size-10 shrink-0 items-center justify-center rounded-full bg-slate-200 text-sm font-semibold text-slate-700 dark:bg-slate-700 dark:text-slate-200">{{ conversation.peer.id ? conversation.peer.username.slice(0, 1).toUpperCase() : '—' }}</span>
            <span class="min-w-0 flex-1">
              <span class="flex items-center justify-between gap-2"><span class="truncate font-medium">{{ conversation.peer.id ? conversation.peer.username : 'Conversación finalizada' }}</span><template v-if="conversation.peer.id"><AccountBadge :provider="conversation.peer.authProvider" /><PresenceDot :online="!!chat.onlineUsers[conversation.peer.id]" /></template><span v-else class="text-[10px] text-slate-500 dark:text-slate-400">Finalizada</span></span>
              <span class="mt-0.5 block truncate text-xs text-slate-500 dark:text-slate-400">{{ conversation.lastMessage?.content ?? 'Sin mensajes' }}</span>
            </span>
            <span v-if="conversation.unreadCount" class="rounded-full bg-sky-600 px-2 py-0.5 text-xs text-white" aria-label="Mensajes sin leer">{{ conversation.unreadCount }}</span>
          </button>
        </li>
      </ul>
      <p v-else class="p-4 text-sm text-slate-500 dark:text-slate-400">Aún no tienes chats con otras personas. Selecciona «Nuevo chat» para comenzar.</p>
    </template>
  </aside>
</template>
