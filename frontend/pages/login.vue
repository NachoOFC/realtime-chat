<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'
import { useApi } from '~/composables/useApi'
const auth = useAuthStore()
const config = useRuntimeConfig()
const route = useRoute()
const username = ref('')
const errorMessage = ref<string | null>(null)
const googleLoading = ref(false)
const { data: providers } = await useAsyncData('auth-providers', () => useApi<{ google: boolean }>('/api/auth/providers'), { default: () => ({ google: false }) })
const oauthMessage = computed(() => {
  const messages: Record<string, string> = {
    google_cancelled: 'Cancelaste el acceso con Google. Puedes intentarlo otra vez.',
    google_state: 'El acceso con Google venció o no es válido. Intenta de nuevo.',
    google_failed: 'No se pudo verificar tu cuenta de Google. Intenta de nuevo.',
    google_unavailable: 'El acceso con Google todavía no está disponible.',
    google_email_in_use: 'Ya existe una cuenta con ese correo. Usa su método de acceso original.',
  }
  return typeof route.query.authError === 'string' ? messages[route.query.authError] : undefined
})
function continueWithGoogle() {
  if (!providers.value.google || googleLoading.value) return
  googleLoading.value = true
  window.location.assign(new URL('/api/auth/google', config.public.apiBase || window.location.origin).href)
}
async function submit() {
  errorMessage.value = null
  try { await auth.enterAsGuest(username.value); await navigateTo('/chat') }
  catch (error) { errorMessage.value = readError(error) }
}
function readError(error: unknown) {
  if (typeof error === 'object' && error !== null && 'data' in error) {
    const data = (error as { data?: { error?: string } }).data
    if (data?.error) return data.error
  }
  return 'No se pudo entrar. Intenta de nuevo.'
}
</script>

<template>
  <section class="mx-auto flex w-full max-w-md flex-1 flex-col justify-center py-10">
    <div class="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm dark:border-slate-700 dark:bg-slate-900">
      <h1 class="text-2xl font-semibold tracking-tight">Entra al chat</h1>
      <p class="mt-2 text-sm text-slate-600 dark:text-slate-300">Con Google puedes volver a tu cuenta y conservar tus conversaciones.</p>
      <button type="button" :disabled="!providers.google || googleLoading" class="mt-5 flex w-full items-center justify-center gap-3 rounded-lg border border-slate-300 bg-white px-4 py-2.5 font-medium text-slate-700 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-50" @click="continueWithGoogle">
        <svg aria-hidden="true" viewBox="0 0 24 24" class="size-5"><path fill="#4285F4" d="M21.6 12.23c0-.71-.06-1.39-.18-2.05H12v3.88h5.38a4.6 4.6 0 0 1-1.99 3.02v2.51h3.23c1.89-1.74 2.98-4.31 2.98-7.36Z"/><path fill="#34A853" d="M12 22c2.7 0 4.96-.9 6.62-2.41l-3.23-2.51c-.9.6-2.04.97-3.39.97-2.6 0-4.81-1.76-5.6-4.12H3.06v2.59A10 10 0 0 0 12 22Z"/><path fill="#FBBC05" d="M6.4 13.93a6 6 0 0 1 0-3.86V7.48H3.06a10 10 0 0 0 0 9.04l3.34-2.59Z"/><path fill="#EA4335" d="M12 5.95c1.47 0 2.79.51 3.82 1.5l2.86-2.87A9.6 9.6 0 0 0 12 2a10 10 0 0 0-8.94 5.48l3.34 2.59C7.19 7.71 9.4 5.95 12 5.95Z"/></svg>
        {{ googleLoading ? 'Abriendo Google...' : 'Continuar con Google' }}
      </button>
      <p v-if="!providers.google" class="mt-2 text-xs text-slate-500 dark:text-slate-400">Google aún no está disponible. Puedes entrar como invitado.</p>
      <p v-if="oauthMessage" role="alert" class="mt-3 rounded-lg bg-red-50 px-3 py-2 text-sm text-red-700 dark:bg-red-950 dark:text-red-300">{{ oauthMessage }}</p>
      <div class="my-6 flex items-center gap-3 text-xs text-slate-500 dark:text-slate-400"><span class="h-px flex-1 bg-slate-200 dark:bg-slate-700" />o entra como invitado<span class="h-px flex-1 bg-slate-200 dark:bg-slate-700" /></div>
      <h2 class="text-lg font-semibold">Elige tu apodo</h2>
      <p class="mt-1 text-sm text-slate-500 dark:text-slate-400">Sin correo ni contraseña. Podrás hablar con todos y ver quién está conectado.</p>
      <form class="mt-4 space-y-4" @submit.prevent="submit">
        <div>
          <label for="username" class="text-sm font-medium text-slate-700 dark:text-slate-200">Apodo de invitado</label>
          <input id="username" v-model="username" type="text" autocomplete="nickname" minlength="3" maxlength="20" required placeholder="Ej: Juanito" class="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 outline-none focus:border-slate-500 dark:border-slate-600 dark:bg-slate-900 dark:focus:border-sky-400">
        </div>
        <p v-if="errorMessage" role="alert" class="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-700 dark:bg-red-950 dark:text-red-300">{{ errorMessage }}</p>
        <button type="submit" :disabled="auth.loading" class="w-full rounded-lg bg-slate-900 px-4 py-2 font-medium text-white transition hover:bg-slate-800 disabled:opacity-50 dark:bg-sky-600 dark:hover:bg-sky-500">{{ auth.loading ? 'Entrando...' : 'Entrar como invitado' }}</button>
      </form>
      <p class="mt-4 rounded-lg bg-amber-50 p-3 text-xs leading-relaxed text-amber-900 dark:bg-amber-950 dark:text-amber-200"><strong>El acceso de invitado es temporal.</strong> Al pulsar «Salir» se eliminan tu perfil y los mensajes que enviaste, también del historial de otras personas. No podrás recuperarlos. Si abandonas la sesión, se eliminarán al vencer tu acceso.</p>
    </div>
  </section>
</template>
