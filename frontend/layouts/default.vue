<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'
import { useDemoMode } from '~/composables/useDemoMode'

const auth = useAuthStore()
const demoMode = useDemoMode()
</script>

<template>
  <div class="flex min-h-screen flex-col">
    <header class="border-b border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900">
      <div class="mx-auto flex w-full max-w-6xl items-center justify-between gap-4 px-4 py-3">
        <NuxtLink to="/chat" class="text-lg font-semibold tracking-tight">
          Realtime Chat
        </NuxtLink>

        <div class="flex items-center gap-3 text-sm">
          <ThemeToggle />
          <template v-if="auth.user">
          <span
            v-if="demoMode"
            class="rounded-full bg-amber-100 dark:bg-amber-950 px-2.5 py-1 text-xs font-semibold text-amber-800 dark:text-amber-200"
          >
            Modo demo
          </span>
          <span class="hidden text-slate-600 dark:text-slate-300 sm:inline">{{ auth.user.username }}</span>
          <button
            v-if="!demoMode"
            type="button"
            class="rounded-lg border border-slate-300 dark:border-slate-600 px-3 py-1.5 font-medium text-slate-700 dark:text-slate-200 transition hover:bg-slate-50 dark:hover:bg-slate-800"
            @click="auth.logout()"
          >
            Salir
          </button>
          </template>
        </div>
      </div>
    </header>

    <main class="mx-auto flex w-full max-w-6xl flex-1 flex-col px-4 py-4">
      <slot />
    </main>
  </div>
</template>
