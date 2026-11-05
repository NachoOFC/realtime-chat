<script setup lang="ts">
import { useChatStore } from '~/stores/chat'
import { useAssistantStore } from '~/stores/assistant'
import { ASSISTANT_CONVERSATION_ID } from '~/utils/assistant'
import { conversationTitle } from '~/utils/conversations'
const chat = useChatStore()
const assistant = useAssistantStore()
const showUsers = ref(false)
const groupMode = ref(false)
const groupName = ref('')
const selected = ref<string[]>([])
const search = ref('')
const starting = ref(false)
const onlyOnline = ref(false)
const filteredUsers = computed(() => chat.users.filter(user => (!onlyOnline.value || chat.onlineUsers[user.id]) && user.username.toLocaleLowerCase().includes(search.value.trim().toLocaleLowerCase())).sort((a, b) => Number(!!chat.onlineUsers[b.id]) - Number(!!chat.onlineUsers[a.id]) || a.username.localeCompare(b.username)))
async function openUsers() { showUsers.value = true; groupMode.value = false; selected.value = []; groupName.value = ''; await chat.loadUsers() }
async function start(id: string) {
  if (starting.value) return
  starting.value = true
  try { if (await chat.startConversation(id)) showUsers.value = false }
  finally { starting.value = false }
}
async function createGroup() {
  if (starting.value || !selected.value.length || !groupName.value.trim()) return
  starting.value = true
  try { if (await chat.createGroup(groupName.value, selected.value)) { showUsers.value = false; selected.value = []; groupName.value = '' } }
  finally { starting.value = false }
}
</script>

<template>
  <aside class="flex min-h-0 w-80 shrink-0 flex-col overflow-hidden border-r border-slate-200 bg-slate-50/60 dark:border-slate-700 dark:bg-slate-950/60 max-md:w-full" :class="{ 'max-md:hidden': chat.activeConversationId }">
    <div class="flex shrink-0 items-center justify-between gap-2 border-b border-slate-200 px-4 py-3 dark:border-slate-700">
      <h2 class="text-sm font-medium">{{ showUsers ? 'Nuevo chat' : 'Chats' }}</h2>
      <button type="button" class="rounded-md px-2 py-1 text-sm text-slate-600 hover:bg-slate-200 dark:text-slate-300 dark:hover:bg-slate-800" @click="showUsers ? showUsers = false : openUsers()">{{ showUsers ? 'Volver' : 'Nuevo chat' }}</button>
    </div>
    <div v-if="showUsers" class="flex min-h-0 flex-col gap-2 overflow-y-auto overscroll-contain p-3">
      <div class="mb-2 flex gap-4 border-b border-slate-200 text-sm dark:border-slate-700">
        <button type="button" class="border-b-2 px-1 pb-2" :class="!groupMode ? 'border-slate-800 dark:border-slate-200' : 'border-transparent text-slate-500'" @click="groupMode = false">Persona</button>
        <button type="button" class="border-b-2 px-1 pb-2" :class="groupMode ? 'border-slate-800 dark:border-slate-200' : 'border-transparent text-slate-500'" @click="groupMode = true">Nuevo grupo</button>
      </div>
      <template v-if="groupMode">
        <label for="group-name" class="text-sm">Nombre del grupo</label>
        <input id="group-name" v-model="groupName" maxlength="60" placeholder="Nombre" class="mb-2 rounded-md border border-slate-300 bg-white px-3 py-2 text-sm dark:border-slate-600 dark:bg-slate-900">
      </template>
      <label for="user-search" class="text-sm">Buscar por apodo</label>
      <input id="user-search" v-model="search" type="search" placeholder="Escribe un apodo" class="rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm dark:border-slate-600 dark:bg-slate-900">
      <button type="button" class="self-start text-xs text-sky-700 dark:text-sky-300" :disabled="chat.loadingUsers" @click="chat.loadUsers()">Actualizar usuarios</button>
      <label class="flex items-center gap-2 text-xs"><input v-model="onlyOnline" type="checkbox">Solo conectados</label>
      <p v-if="chat.loadingUsers" class="text-sm text-slate-500 dark:text-slate-400">Cargando usuarios...</p>
      <ul v-else>
        <li v-for="user in filteredUsers" :key="user.id">
          <label v-if="groupMode" class="flex items-center gap-2 px-2 py-3 text-sm"><input v-model="selected" type="checkbox" :value="user.id" :disabled="starting || (!selected.includes(user.id) && selected.length >= 49)"><span class="min-w-0 flex-1 truncate">{{ user.username }}</span><AccountBadge :provider="user.authProvider" /></label>
          <button v-else type="button" :disabled="starting" class="flex w-full items-center gap-2 rounded-md px-3 py-3 text-left hover:bg-white disabled:opacity-50 dark:hover:bg-slate-800" @click="start(user.id)"><PresenceDot :online="!!chat.onlineUsers[user.id]" /><span class="min-w-0 flex-1 truncate">{{ user.username }}</span><AccountBadge :provider="user.authProvider" /></button>
        </li>
      </ul>
      <p v-if="!chat.loadingUsers && !filteredUsers.length" class="text-sm text-slate-500 dark:text-slate-400">No hay personas disponibles con ese apodo.</p>
      <button v-if="groupMode" type="button" :disabled="starting || !selected.length || !groupName.trim()" class="mt-2 rounded-md bg-slate-900 px-3 py-2 text-sm text-white disabled:opacity-50 dark:bg-slate-200 dark:text-slate-900" @click="createGroup">{{ starting ? 'Creando…' : `Crear grupo (${selected.length + 1})` }}</button>
    </div>
    <template v-else>
      <button type="button" class="flex w-full shrink-0 items-center gap-3 border-b border-slate-200 px-4 py-4 text-left transition hover:bg-white dark:border-slate-700 dark:hover:bg-slate-800" :class="{ 'bg-white dark:bg-slate-900': chat.activeConversationId === ASSISTANT_CONVERSATION_ID }" :aria-pressed="chat.activeConversationId === ASSISTANT_CONVERSATION_ID" @click="chat.selectConversation(ASSISTANT_CONVERSATION_ID)">
        <span class="flex size-10 shrink-0 items-center justify-center rounded-full bg-slate-200 text-xs font-medium text-slate-700 dark:bg-slate-700 dark:text-slate-200" aria-hidden="true">IA</span>
        <span class="min-w-0 flex-1">
          <span class="block font-medium">IACHAT</span>
          <span class="mt-0.5 block truncate text-xs text-slate-500 dark:text-slate-400">{{ assistant.sending ? 'Escribiendo…' : assistant.messages.at(-1)?.content || 'Asistente' }}</span>
        </span>
      </button>
      <p v-if="chat.loadingConversations" class="p-4 text-sm text-slate-500 dark:text-slate-400">Cargando...</p>
      <ul v-else-if="chat.conversations.length" class="min-h-0 flex-1 overflow-y-auto overscroll-contain">
        <li v-for="conversation in chat.conversations" :key="conversation.id">
          <button type="button" class="flex w-full items-center gap-3 px-4 py-3 text-left transition hover:bg-white dark:hover:bg-slate-800" :class="{ 'bg-white dark:bg-slate-900': chat.activeConversationId === conversation.id }" @click="chat.selectConversation(conversation.id)">
            <span class="flex size-10 shrink-0 items-center justify-center rounded-full bg-slate-200 text-sm font-medium text-slate-700 dark:bg-slate-700 dark:text-slate-200">{{ conversationTitle(conversation).slice(0, 1).toUpperCase() }}</span>
            <span class="min-w-0 flex-1">
              <span class="flex items-center justify-between gap-2"><span class="truncate font-medium">{{ conversationTitle(conversation) }}</span><span v-if="conversation.kind === 'group'" class="shrink-0 text-[10px] text-slate-500 dark:text-slate-400">{{ conversation.members?.length }} personas</span><template v-else-if="conversation.peer.id"><AccountBadge :provider="conversation.peer.authProvider" /><PresenceDot :online="!!chat.onlineUsers[conversation.peer.id]" /></template><span v-else class="text-[10px] text-slate-500 dark:text-slate-400">Finalizada</span></span>
              <span class="mt-0.5 block truncate text-xs text-slate-500 dark:text-slate-400">{{ conversation.kind === 'group' && conversation.lastMessage?.senderName ? `${conversation.lastMessage.senderName}: ` : '' }}{{ conversation.lastMessage?.content ?? 'Sin mensajes' }}</span>
            </span>
            <span v-if="conversation.unreadCount" class="rounded-full bg-sky-600 px-2 py-0.5 text-xs text-white" aria-label="Mensajes sin leer">{{ conversation.unreadCount }}</span>
          </button>
        </li>
      </ul>
      <p v-else class="p-4 text-sm text-slate-500 dark:text-slate-400">Aún no tienes chats con otras personas. Selecciona «Nuevo chat» para comenzar.</p>
    </template>
  </aside>
</template>
