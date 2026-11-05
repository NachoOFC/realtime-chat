<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'

const auth = useAuthStore()
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
  <div class="flex h-dvh flex-col overflow-hidden">
    <header class="shrink-0 border-b border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900">
      <div class="mx-auto flex w-full max-w-6xl items-center justify-between gap-4 px-4 py-3">
        <NuxtLink to="/chat" class="text-lg font-semibold tracking-tight">
          Realtime Chat
        </NuxtLink>

        <div class="flex items-center gap-3 text-sm">
          <ThemeToggle />
          <template v-if="auth.user">
          <span class="hidden text-slate-600 dark:text-slate-300 sm:inline">{{ auth.user.username }}</span>
          <AccountBadge :provider="auth.user.authProvider" />
          <button
            type="button"
            class="rounded-lg border border-slate-300 dark:border-slate-600 px-3 py-1.5 font-medium text-slate-700 dark:text-slate-200 transition hover:bg-slate-50 dark:hover:bg-slate-800"
            @click="logout"
            :disabled="loggingOut"
          >
            {{ loggingOut ? 'Saliendo...' : 'Salir' }}
          </button>
          </template>
        </div>
      </div>
    </header>

    <main class="mx-auto flex min-h-0 w-full max-w-6xl flex-1 flex-col overflow-y-auto px-4 py-4">
      <p v-if="logoutError" role="alert" class="mb-3 rounded-lg bg-red-50 p-3 text-sm text-red-700 dark:bg-red-950 dark:text-red-300">{{ logoutError }}</p>
      <slot />
    </main>
  </div>
</template>
