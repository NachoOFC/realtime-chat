<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'

const auth = useAuthStore()

const mode = ref<'login' | 'register'>('login')
const email = ref('')
const username = ref('')
const password = ref('')
const errorMessage = ref<string | null>(null)

async function submit() {
  errorMessage.value = null

  try {
    if (mode.value === 'login') {
      await auth.login(email.value, password.value)
    } else {
      await auth.register(email.value, username.value, password.value)
    }

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

  return 'No se pudo completar la operación.'
}
</script>

<template>
  <section class="mx-auto flex w-full max-w-md flex-1 flex-col justify-center py-10">
    <div class="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
      <h1 class="text-2xl font-semibold tracking-tight">
        {{ mode === 'login' ? 'Iniciar sesión' : 'Crear cuenta' }}
      </h1>
      <p class="mt-1 text-sm text-slate-500">
        Sesión segura con cookie HttpOnly. Nada se guarda en localStorage.
      </p>

      <form class="mt-6 space-y-4" @submit.prevent="submit">
        <div>
          <label for="email" class="text-sm font-medium text-slate-700">Email</label>
          <input
            id="email"
            v-model="email"
            type="email"
            autocomplete="email"
            required
            class="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 outline-none focus:border-slate-500"
          >
        </div>

        <div v-if="mode === 'register'">
          <label for="username" class="text-sm font-medium text-slate-700">Usuario</label>
          <input
            id="username"
            v-model="username"
            type="text"
            autocomplete="username"
            minlength="3"
            maxlength="32"
            required
            class="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 outline-none focus:border-slate-500"
          >
        </div>

        <div>
          <label for="password" class="text-sm font-medium text-slate-700">Contraseña</label>
          <input
            id="password"
            v-model="password"
            type="password"
            :autocomplete="mode === 'login' ? 'current-password' : 'new-password'"
            minlength="8"
            required
            class="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 outline-none focus:border-slate-500"
          >
        </div>

        <p v-if="errorMessage" class="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-700">
          {{ errorMessage }}
        </p>

        <button
          type="submit"
          :disabled="auth.loading"
          class="w-full rounded-lg bg-slate-900 px-4 py-2 font-medium text-white transition hover:bg-slate-800 disabled:opacity-50"
        >
          {{ auth.loading ? 'Procesando...' : mode === 'login' ? 'Entrar' : 'Registrarme' }}
        </button>
      </form>

      <button
        type="button"
        class="mt-4 w-full text-sm text-slate-600 underline-offset-4 hover:underline"
        @click="mode = mode === 'login' ? 'register' : 'login'"
      >
        {{ mode === 'login' ? '¿No tienes cuenta? Crear una' : 'Ya tengo cuenta' }}
      </button>
    </div>
  </section>
</template>
