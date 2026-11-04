<script setup lang="ts">
defineProps<{ provider?: 'google' | 'guest' | 'password' }>()
</script>
<template>
  <span v-if="provider === 'google'" class="inline-flex shrink-0 items-center gap-1 rounded-full bg-sky-100 px-2 py-0.5 text-[10px] font-medium text-sky-800 dark:bg-sky-950 dark:text-sky-200" title="Esta persona inició sesión con una cuenta de Google"><span aria-hidden="true" class="font-bold">G</span> Google</span>
  <span v-else-if="provider === 'guest'" class="inline-flex shrink-0 rounded-full bg-amber-100 px-2 py-0.5 text-[10px] font-medium text-amber-800 dark:bg-amber-950 dark:text-amber-200">Invitado</span>
  <span v-else-if="provider === 'password'" class="text-[10px] text-slate-500 dark:text-slate-400">Cuenta</span>
</template>
