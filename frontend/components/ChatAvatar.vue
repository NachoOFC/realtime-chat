<script setup lang="ts">
const props = withDefaults(defineProps<{ name: string; image?: string; online?: boolean; group?: boolean; assistant?: boolean; size?: 'sm' | 'md' | 'lg' }>(), { size: 'md', online: undefined })
const failed = ref(false)
watch(() => props.image, () => { failed.value = false })
const initials = computed(() => props.name.trim().split(/\s+/).slice(0, 2).map(part => part[0]).join('').toUpperCase())
</script>
<template>
  <span class="relative inline-flex shrink-0" :class="size === 'sm' ? 'size-8' : size === 'lg' ? 'size-12' : 'size-10'">
    <span class="flex size-full items-center justify-center overflow-hidden rounded-full bg-[#eef2ff] text-sm font-medium text-[#5260a5] dark:bg-slate-800 dark:text-slate-300">
      <img v-if="image && !failed" :src="image" :alt="name" class="size-full object-cover" referrerpolicy="no-referrer" @error="failed = true">
      <ChatIcon v-else-if="group" name="users" :size="size === 'sm' ? 16 : 20" />
      <ChatIcon v-else-if="assistant" name="assistant" :size="size === 'sm' ? 16 : 20" />
      <span v-else>{{ initials || '?' }}</span>
    </span>
    <span v-if="online !== undefined" class="absolute bottom-0 right-0 size-3 rounded-full border-2 border-white dark:border-slate-900" :class="online ? 'bg-emerald-500' : 'bg-slate-400'" :title="online ? 'En línea' : 'Desconectado'" />
  </span>
</template>
