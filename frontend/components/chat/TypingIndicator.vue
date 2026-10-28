<script setup lang="ts">
defineProps<{
  active: boolean
}>()
</script>

<template>
  <div v-if="active" class="flex items-center gap-1 px-4 py-2 text-xs text-slate-500 dark:text-slate-400">
    <span class="size-1.5 animate-bounce rounded-full bg-slate-400" />
    <span class="size-1.5 animate-bounce rounded-full bg-slate-400 [animation-delay:120ms]" />
    <span class="size-1.5 animate-bounce rounded-full bg-slate-400 [animation-delay:240ms]" />
    <span class="ml-1">escribiendo...</span>
  </div>
</template>
