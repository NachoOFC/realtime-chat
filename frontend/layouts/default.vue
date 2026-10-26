<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'

const auth = useAuthStore()
</script>

<template>
  <div class="flex min-h-screen flex-col">
    <header class="border-b border-slate-200 bg-white">
      <div class="mx-auto flex w-full max-w-6xl items-center justify-between gap-4 px-4 py-3">
        <NuxtLink to="/chat" class="text-lg font-semibold tracking-tight">
          Realtime Chat
        </NuxtLink>

        <div v-if="auth.user" class="flex items-center gap-3 text-sm">
          <span class="hidden text-slate-600 sm:inline">{{ auth.user.username }}</span>
          <button
            type="button"
            class="rounded-lg border border-slate-300 px-3 py-1.5 font-medium text-slate-700 transition hover:bg-slate-50"
            @click="auth.logout()"
          >
            Salir
          </button>
        </div>
      </div>
    </header>

    <main class="mx-auto flex w-full max-w-6xl flex-1 flex-col px-4 py-4">
      <slot />
    </main>
  </div>
</template>
