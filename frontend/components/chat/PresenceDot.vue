<script setup lang="ts">
defineProps<{
  online: boolean
}>()
</script>

<template>
  <span
    class="inline-block size-2 rounded-full"
    :class="online ? 'bg-emerald-500' : 'bg-slate-300'"
  />
</template>
