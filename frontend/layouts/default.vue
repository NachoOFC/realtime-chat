<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'
import { useChatStore } from '~/stores/chat'
import { ASSISTANT_CONVERSATION_ID } from '~/utils/assistant'

const auth = useAuthStore()
const chat = useChatStore()
function showChats(filter: 'all' | 'groups') { chat.conversationFilter = filter; chat.activeConversationId = null; void navigateTo('/chat') }
function showAssistant() { chat.activeConversationId = ASSISTANT_CONVERSATION_ID; void navigateTo('/chat') }
onMounted(() => auth.rememberCurrentAccount())
const logoutError = ref('')
const loggingOut = ref(false)
async function logout() {
  if (auth.user?.isGuest && !window.confirm('Al salir se eliminarán tu perfil de invitado y todos los mensajes que enviaste. No se pueden recuperar. ¿Quieres salir?')) return
  loggingOut.value = true
  logoutError.value = ''
  try { await auth.logout() }
  catch { logoutError.value = 'No se pudo cerrar la sesión. Intenta de nuevo.' }
  finally { loggingOut.value = false }
}
</script>

<template>
  <div class="flex h-dvh overflow-hidden bg-[#f9fafb] dark:bg-[#101828]">
    <aside v-if="auth.user" class="hidden w-20 shrink-0 flex-col border-r border-slate-200 bg-white px-3 py-6 dark:border-slate-800 dark:bg-slate-900 lg:flex xl:w-56 xl:px-5">
      <NuxtLink to="/chat" class="mb-10 flex items-center gap-3 px-1"><span class="flex size-9 shrink-0 items-center justify-center rounded-xl bg-[#465fff] text-white"><ChatIcon name="chat" :size="23" /></span><span class="hidden text-lg font-semibold tracking-tight xl:block">Realtime Chat</span></NuxtLink>
      <p class="mb-3 hidden px-3 text-[11px] uppercase tracking-wider text-slate-400 xl:block">Espacio de chat</p>
      <nav class="space-y-1.5" aria-label="Navegación principal">
        <button type="button" class="flex w-full items-center justify-center gap-3 rounded-lg px-3 py-3 text-sm xl:justify-start" :class="chat.conversationFilter === 'all' && chat.activeConversationId !== ASSISTANT_CONVERSATION_ID ? 'bg-[#ecf3ff] text-[#465fff] dark:bg-[#465fff]/10' : 'text-slate-500 hover:bg-slate-50 dark:hover:bg-slate-800'" title="Todos los chats" @click="showChats('all')"><ChatIcon name="chat" /><span class="hidden xl:block">Mensajes</span></button>
        <button type="button" class="flex w-full items-center justify-center gap-3 rounded-lg px-3 py-3 text-sm xl:justify-start" :class="chat.conversationFilter === 'groups' && chat.activeConversationId !== ASSISTANT_CONVERSATION_ID ? 'bg-[#ecf3ff] text-[#465fff] dark:bg-[#465fff]/10' : 'text-slate-500 hover:bg-slate-50 dark:hover:bg-slate-800'" title="Grupos" @click="showChats('groups')"><ChatIcon name="users" /><span class="hidden xl:block">Grupos</span></button>
        <button type="button" class="flex w-full items-center justify-center gap-3 rounded-lg px-3 py-3 text-sm xl:justify-start" :class="chat.activeConversationId === ASSISTANT_CONVERSATION_ID ? 'bg-[#ecf3ff] text-[#465fff] dark:bg-[#465fff]/10' : 'text-slate-500 hover:bg-slate-50 dark:hover:bg-slate-800'" title="IACHAT" @click="showAssistant"><ChatIcon name="assistant" /><span class="hidden xl:block">IACHAT</span></button>
      </nav>
    </aside>
    <div class="flex min-w-0 flex-1 flex-col overflow-hidden">
    <header class="shrink-0 border-b border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900">
      <div class="flex w-full items-center justify-between gap-4 px-4 py-4 lg:px-6">
        <NuxtLink to="/chat" class="text-base font-semibold tracking-tight" :class="auth.user ? 'lg:hidden' : ''">
          Realtime Chat
        </NuxtLink>
        <p v-if="auth.user" class="hidden text-sm text-slate-400 lg:block">Tu espacio <span class="mx-2 text-slate-300">/</span> <span class="text-slate-700 dark:text-slate-200">Mensajes</span></p>

        <div class="flex items-center gap-3 text-sm">
          <ThemeToggle />
          <template v-if="auth.user">
          <ChatAvatar :name="auth.user.username" :image="auth.user.avatarUrl" size="sm" />
          <div class="hidden sm:block"><p class="text-sm font-medium">{{ auth.user.username }}</p><AccountBadge :provider="auth.user.authProvider" /></div>
          <button
            type="button"
            class="rounded-lg border border-slate-200 px-3 py-2 text-xs font-medium text-slate-500 transition hover:bg-slate-50 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800"
            @click="logout"
            :disabled="loggingOut"
          >
            {{ loggingOut ? 'Saliendo...' : 'Salir' }}
          </button>
          </template>
        </div>
      </div>
    </header>

    <main class="flex min-h-0 w-full flex-1 flex-col overflow-y-auto px-4 py-4 lg:p-6">
      <p v-if="logoutError" role="alert" class="mb-3 rounded-lg bg-red-50 p-3 text-sm text-red-700 dark:bg-red-950 dark:text-red-300">{{ logoutError }}</p>
      <slot />
    </main>
    </div>
  </div>
</template>
