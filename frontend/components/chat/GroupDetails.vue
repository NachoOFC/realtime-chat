<script setup lang="ts">
import type { Conversation } from '~/types'
import { useChatStore } from '~/stores/chat'
import { useAuthStore } from '~/stores/auth'

const props = defineProps<{ conversation: Conversation }>()
const chat = useChatStore()
const auth = useAuthStore()
const adding = ref(false)
const busy = ref(false)
const search = ref('')
const selected = ref<string[]>([])
const available = computed(() => {
  const ids = new Set(props.conversation.members?.map(member => member.id))
  return chat.users.filter(user => !ids.has(user.id) && user.username.toLocaleLowerCase().includes(search.value.trim().toLocaleLowerCase()))
})
async function showAdd() { adding.value = !adding.value; if (adding.value) await chat.loadUsers() }
async function add() {
  if (busy.value || !selected.value.length) return
  busy.value = true
  try { if (await chat.addGroupMembers(props.conversation.id, selected.value)) { selected.value = []; adding.value = false } }
  finally { busy.value = false }
}
async function leave() {
  if (busy.value || !window.confirm(`¿Salir de «${props.conversation.name}»? Dejarás de recibir mensajes de este grupo.`)) return
  busy.value = true
  try { await chat.leaveGroup(props.conversation.id) } finally { busy.value = false }
}
</script>

<template>
  <div class="max-h-[45%] shrink-0 overflow-y-auto border-b border-slate-200 bg-slate-50 px-4 py-3 text-sm dark:border-slate-700 dark:bg-slate-950" aria-label="Participantes del grupo">
    <div class="flex items-center justify-between gap-3">
      <h2 class="font-medium">Personas en el grupo</h2>
      <button type="button" :disabled="busy || (conversation.members?.length || 0) >= 50" class="text-sm text-slate-600 underline underline-offset-4 disabled:opacity-50 dark:text-slate-300" @click="showAdd">{{ adding ? 'Cancelar' : 'Agregar personas' }}</button>
    </div>
    <ul v-if="!adding" class="mt-2 divide-y divide-slate-200 dark:divide-slate-800">
      <li v-for="member in conversation.members" :key="member.id" class="flex items-center gap-2 py-2">
        <PresenceDot :online="!!chat.onlineUsers[member.id]" />
        <span class="min-w-0 flex-1 truncate">{{ member.username }}<span v-if="member.id === auth.user?.id" class="text-slate-500 dark:text-slate-400"> (tú)</span></span>
        <AccountBadge :provider="member.authProvider" />
      </li>
    </ul>
    <form v-else class="mt-3 space-y-3" @submit.prevent="add">
      <input v-model="search" type="search" aria-label="Buscar personas para el grupo" placeholder="Buscar por apodo" class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 dark:border-slate-600 dark:bg-slate-900">
      <p v-if="chat.loadingUsers" class="text-slate-500">Cargando personas…</p>
      <ul v-else>
        <li v-for="user in available" :key="user.id"><label class="flex items-center gap-2 py-2"><input v-model="selected" type="checkbox" :value="user.id" :disabled="busy || (!selected.includes(user.id) && selected.length >= 50 - (conversation.members?.length || 0))"><span class="min-w-0 flex-1 truncate">{{ user.username }}</span><AccountBadge :provider="user.authProvider" /></label></li>
      </ul>
      <p v-if="!chat.loadingUsers && !available.length" class="text-slate-500 dark:text-slate-400">No hay más personas disponibles.</p>
      <button type="submit" :disabled="busy || !selected.length" class="rounded-md bg-slate-900 px-3 py-2 text-white disabled:opacity-50 dark:bg-slate-200 dark:text-slate-900">{{ busy ? 'Agregando…' : `Agregar (${selected.length})` }}</button>
    </form>
    <button v-if="!adding" type="button" :disabled="busy" class="mt-3 text-sm text-red-700 disabled:opacity-50 dark:text-red-400" @click="leave">{{ busy ? 'Saliendo…' : 'Salir del grupo' }}</button>
  </div>
</template>
