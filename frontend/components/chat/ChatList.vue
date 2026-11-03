<script setup lang="ts">
import { useChatStore } from '~/stores/chat'
const chat = useChatStore()
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
      <p v-if="chat.loadingConversations" class="p-4 text-sm text-slate-500 dark:text-slate-400">Cargando...</p>
      <ul v-else-if="chat.conversations.length" class="flex-1 overflow-y-auto">
        <li v-for="conversation in chat.conversations" :key="conversation.id">
          <button type="button" class="flex w-full items-center gap-3 px-4 py-3 text-left transition hover:bg-white dark:hover:bg-slate-800" :class="{ 'bg-white dark:bg-slate-900': chat.activeConversationId === conversation.id }" @click="chat.selectConversation(conversation.id)">
            <span class="flex size-10 shrink-0 items-center justify-center rounded-full bg-slate-200 text-sm font-semibold text-slate-700 dark:bg-slate-700 dark:text-slate-200">{{ conversation.peer.username.slice(0, 1).toUpperCase() }}</span>
            <span class="min-w-0 flex-1">
              <span class="flex items-center justify-between gap-2"><span class="truncate font-medium">{{ conversation.peer.username }}</span><AccountBadge :provider="conversation.peer.authProvider" /><PresenceDot :online="!!chat.onlineUsers[conversation.peer.id]" /></span>
              <span class="mt-0.5 block truncate text-xs text-slate-500 dark:text-slate-400">{{ conversation.lastMessage?.content ?? 'Sin mensajes' }}</span>
            </span>
            <span v-if="conversation.unreadCount" class="rounded-full bg-sky-600 px-2 py-0.5 text-xs text-white" aria-label="Mensajes sin leer">{{ conversation.unreadCount }}</span>
          </button>
        </li>
      </ul>
      <p v-else class="p-4 text-sm text-slate-500 dark:text-slate-400">Aún no hay conversaciones. Selecciona «Nuevo chat» para hablar con alguien.</p>
    </template>
  </aside>
</template>
