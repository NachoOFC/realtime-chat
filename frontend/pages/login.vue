<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'

const auth = useAuthStore()

const username = ref('')
const errorMessage = ref<string | null>(null)

async function submit() {
  errorMessage.value = null

  try {
    await auth.enterAsGuest(username.value)
    await navigateTo('/chat')
  } catch (error) {
    errorMessage.value = readError(error)
  }
}

function readError(error: unknown) {
  if (typeof error === 'object' && error !== null && 'data' in error) {
    const data = (error as { data?: { error?: string } }).data
    if (data?.error) {
      return data.error
    }
  }

  return 'No se pudo entrar. Probá de nuevo.'
}
</script>

<template>
  <section class="mx-auto flex w-full max-w-md flex-1 flex-col justify-center py-10">
    <div class="rounded-2xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 p-6 shadow-sm">
      <h1 class="text-2xl font-semibold tracking-tight">Elegí tu apodo</h1>
      <p class="mt-1 text-sm text-slate-500 dark:text-slate-400">
        Sin correo ni contraseña. Tu sesión queda guardada en este navegador.
      </p>

      <form class="mt-6 space-y-4" @submit.prevent="submit">
        <div>
          <label for="username" class="text-sm font-medium text-slate-700 dark:text-slate-200">Apodo</label>
          <input
            id="username"
            v-model="username"
            type="text"
            autocomplete="nickname"
            minlength="3"
            maxlength="20"
            autofocus
            required
            placeholder="Ej: lucas_99"
            class="mt-1 w-full rounded-lg border border-slate-300 dark:border-slate-600 px-3 py-2 bg-white dark:bg-slate-900 outline-none focus:border-slate-500 dark:focus:border-sky-400"
          >
        </div>

        <p v-if="errorMessage" class="rounded-lg bg-red-50 dark:bg-red-950 px-3 py-2 text-sm text-red-700 dark:text-red-300">
          {{ errorMessage }}
        </p>

        <button
          type="submit"
          :disabled="auth.loading"
          class="w-full rounded-lg bg-slate-900 dark:bg-sky-600 px-4 py-2 font-medium text-white transition hover:bg-slate-800 dark:hover:bg-sky-500 disabled:opacity-50"
        >
          {{ auth.loading ? 'Entrando...' : 'Entrar' }}
        </button>
      </form>

      <p class="mt-4 text-xs text-slate-400 dark:text-slate-400">
        Si cerrás la sesión o cambiás de navegador, vas a necesitar otro apodo.
      </p>
    </div>
  </section>
</template>
