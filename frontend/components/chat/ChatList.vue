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
const chatSearch = ref('')
const visibleConversations = computed(() => chat.conversations.filter(item => (chat.conversationFilter !== 'groups' || item.kind === 'group') && conversationTitle(item).toLocaleLowerCase().includes(chatSearch.value.trim().toLocaleLowerCase())))
const showAssistant = computed(() => chat.conversationFilter === 'all' && 'iachat asistente gemini'.includes(chatSearch.value.trim().toLocaleLowerCase()))
function messageTime(value?: string) {
  if (!value) return ''
  const date = new Date(value)
  return date.toDateString() === new Date().toDateString() ? date.toLocaleTimeString('es', { hour: '2-digit', minute: '2-digit' }) : date.toLocaleDateString('es', { day: 'numeric', month: 'short' })
}
const starting = ref(false)
const onlyOnline = ref(false)
const filteredUsers = computed(() => chat.users.filter(user => (!onlyOnline.value || chat.onlineUsers[user.id]) && user.username.toLocaleLowerCase().includes(search.value.trim().toLocaleLowerCase())).sort((a, b) => Number(!!chat.onlineUsers[b.id]) - Number(!!chat.onlineUsers[a.id]) || a.username.localeCompare(b.username)))
async function openUsers() { showUsers.value = true; groupMode.value = false; selected.value = []; groupName.value = ''; await chat.loadUsers() }
async function start(id: string) {
  if (starting.value) return
  starting.value = true
  try { if (await chat.startConversation(id)) { showUsers.value = false; chat.conversationFilter = 'all' } }
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
  <aside class="flex min-h-0 w-80 shrink-0 flex-col overflow-hidden rounded-2xl border border-slate-200 bg-white dark:border-slate-800 dark:bg-slate-900 max-md:w-full" :class="{ 'max-md:hidden': chat.activeConversationId }">
    <div class="flex shrink-0 items-center justify-between gap-2 px-5 pb-4 pt-5">
      <h2 class="text-base font-semibold">{{ showUsers ? 'Nuevo mensaje' : 'Mensajes' }}</h2>
      <button type="button" class="flex size-8 items-center justify-center rounded-lg text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-800" :aria-label="showUsers ? 'Volver a mensajes' : 'Nuevo chat'" :title="showUsers ? 'Volver' : 'Nuevo chat'" @click="showUsers ? showUsers = false : openUsers()"><ChatIcon :name="showUsers ? 'back' : 'plus'" :size="19" /></button>
    </div>
    <div v-if="!showUsers" class="shrink-0 px-5 pb-4"><label class="relative block"><ChatIcon name="search" :size="17" class="pointer-events-none absolute left-3 top-3 text-slate-400" /><input v-model="chatSearch" type="search" aria-label="Buscar conversación" placeholder="Buscar conversación…" class="w-full rounded-lg border border-slate-200 bg-transparent py-2.5 pl-9 pr-3 text-sm outline-none placeholder:text-slate-400 focus:border-[#465fff] dark:border-slate-700"></label><div class="mt-3 flex gap-4 text-xs"><button type="button" :class="chat.conversationFilter === 'all' ? 'font-medium text-[#465fff]' : 'text-slate-400'" @click="chat.conversationFilter = 'all'">Todos</button><button type="button" :class="chat.conversationFilter === 'groups' ? 'font-medium text-[#465fff]' : 'text-slate-400'" @click="chat.conversationFilter = 'groups'">Grupos</button></div></div>
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
      <button v-if="showAssistant" type="button" class="mx-3 mb-1 flex shrink-0 items-center gap-3 rounded-xl px-3 py-3 text-left transition hover:bg-[#f2f4f7] dark:hover:bg-slate-800" :class="{ 'bg-[#f2f4f7] dark:bg-slate-800': chat.activeConversationId === ASSISTANT_CONVERSATION_ID }" :aria-pressed="chat.activeConversationId === ASSISTANT_CONVERSATION_ID" @click="chat.selectConversation(ASSISTANT_CONVERSATION_ID)">
        <ChatAvatar name="IACHAT" :assistant="true" />
        <span class="min-w-0 flex-1">
          <span class="flex items-center justify-between gap-2"><span class="text-sm font-medium">IACHAT</span><span class="text-[10px] text-slate-400">Gemini</span></span>
          <span class="mt-0.5 block truncate text-xs text-slate-500 dark:text-slate-400">{{ assistant.sending ? 'Escribiendo…' : assistant.messages.at(-1)?.content || 'Asistente' }}</span>
        </span>
      </button>
      <p v-if="chat.loadingConversations" class="p-4 text-sm text-slate-500 dark:text-slate-400">Cargando...</p>
      <ul v-else-if="visibleConversations.length" class="chat-scroll min-h-0 flex-1 space-y-1 overflow-y-auto overscroll-contain px-3 pb-4">
        <li v-for="conversation in visibleConversations" :key="conversation.id">
          <button type="button" class="flex w-full items-center gap-3 rounded-xl px-3 py-3 text-left transition hover:bg-[#f2f4f7] dark:hover:bg-slate-800" :class="{ 'bg-[#f2f4f7] dark:bg-slate-800': chat.activeConversationId === conversation.id }" @click="chat.selectConversation(conversation.id)">
            <ChatAvatar :name="conversationTitle(conversation)" :image="conversation.kind !== 'group' ? conversation.peer.avatarUrl : undefined" :group="conversation.kind === 'group'" :online="conversation.kind !== 'group' && conversation.peer.id ? !!chat.onlineUsers[conversation.peer.id] : undefined" />
            <span class="min-w-0 flex-1">
              <span class="flex items-center justify-between gap-2"><span class="truncate text-sm font-medium">{{ conversationTitle(conversation) }}</span><span class="shrink-0 text-[10px] text-slate-400">{{ messageTime(conversation.lastMessage?.createdAt || conversation.updatedAt) }}</span></span>
              <span class="mt-0.5 block truncate text-xs text-slate-500 dark:text-slate-400">{{ conversation.kind === 'group' && conversation.lastMessage?.senderName ? `${conversation.lastMessage.senderName}: ` : '' }}{{ conversation.lastMessage?.content ?? 'Sin mensajes' }}</span>
            </span>
            <span v-if="conversation.unreadCount" class="flex min-w-4 items-center justify-center rounded-full bg-[#465fff] px-1 text-[10px] leading-4 text-white" aria-label="Mensajes sin leer">{{ conversation.unreadCount }}</span>
          </button>
        </li>
      </ul>
      <p v-else class="p-5 text-sm leading-6 text-slate-400">{{ chatSearch ? 'No hay conversaciones con ese nombre.' : chat.conversationFilter === 'groups' ? 'Aún no tienes grupos. Crea uno desde el botón +.' : 'Aún no hay conversaciones con otras personas. Usa el botón + para comenzar.' }}</p>
    </template>
  </aside>
</template>
