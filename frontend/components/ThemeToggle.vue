<script setup lang="ts">
const theme = useCookie<'light' | 'dark'>('theme', {
  default: () => 'light',
  maxAge: 60 * 60 * 24 * 365,
  sameSite: 'lax',
})
const isDark = computed(() => theme.value === 'dark')

useHead(() => ({
  htmlAttrs: {
    class: { dark: isDark.value },
    style: { 'color-scheme': isDark.value ? 'dark' : 'light' },
  },
}))
</script>

<template>
  <button
    type="button"
    class="flex size-10 shrink-0 items-center justify-center rounded-lg border border-slate-300 text-slate-700 transition hover:bg-slate-100 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-sky-500 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-800"
    :aria-label="isDark ? 'Activar modo claro' : 'Activar modo oscuro'"
    :title="isDark ? 'Activar modo claro' : 'Activar modo oscuro'"
    :aria-pressed="isDark"
    @click="theme = isDark ? 'light' : 'dark'"
  >
    <svg v-if="isDark" aria-hidden="true" class="size-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
      <circle cx="12" cy="12" r="4" />
      <path d="M12 2v2m0 16v2M2 12h2m16 0h2M4.93 4.93l1.42 1.42m11.3 11.3 1.42 1.42M4.93 19.07l1.42-1.42m11.3-11.3 1.42-1.42" />
    </svg>
    <svg v-else aria-hidden="true" class="size-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
      <path d="M20.9 13A9 9 0 0 1 11 3.1 9 9 0 1 0 20.9 13Z" />
    </svg>
  </button>
</template>
