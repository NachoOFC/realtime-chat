<script setup lang="ts">
withDefaults(defineProps<{ name: string; size?: number }>(), { size: 20 })
const paths: Record<string, string[]> = {
  chat: ['M21 11.5a8.5 8.5 0 0 1-8.5 8.5H4l-2 2V11.5A8.5 8.5 0 0 1 10.5 3h2A8.5 8.5 0 0 1 21 11.5Z', 'M7 10h9M7 14h5'],
  search: ['M21 21l-5-5', 'M18 10.5a7.5 7.5 0 1 1-15 0 7.5 7.5 0 0 1 15 0Z'],
  plus: ['M12 5v14M5 12h14'],
  more: ['M12 5h.01M12 12h.01M12 19h.01'],
  back: ['m14 6-6 6 6 6'],
  send: ['m22 2-7 20-4-9-9-4 20-7ZM22 2 11 13'],
  trash: ['M3 6h18M9 6V4h6v2M5 6l1 15h12l1-15M10 10v7M14 10v7'],
  users: ['M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2M22 21v-2a4 4 0 0 0-3-3.87', 'M13 7a4 4 0 1 1-8 0 4 4 0 0 1 8 0ZM16 3.13a4 4 0 0 1 0 7.75'],
  close: ['m6 6 12 12M18 6 6 18'],
  logout: ['M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4M16 17l5-5-5-5M21 12H9'],
  assistant: ['M12 3v3M8 3h8M5 9h14v11H5V9ZM2 12v5M22 12v5M9 13h.01M15 13h.01M9 17h6'],
  check: ['m5 12 4 4L19 6'],
}
</script>
<template>
  <svg :width="size" :height="size" viewBox="0 0 24 24" fill="none" stroke="currentColor" :stroke-width="name === 'more' ? 3 : 1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path v-for="(d, i) in paths[name] || paths.chat" :key="i" :d="d" /></svg>
</template>
