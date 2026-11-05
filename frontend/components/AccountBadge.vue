<script setup lang="ts">
defineProps<{ provider?: 'google' | 'guest' | 'password' }>()
</script>
<template>
  <span v-if="provider === 'google'" class="inline-flex shrink-0 items-center gap-1 text-[11px] text-slate-500 dark:text-slate-400" title="Esta persona inició sesión con una cuenta de Google"><span aria-hidden="true" class="font-semibold">G</span> Google</span>
  <span v-else-if="provider === 'guest'" class="shrink-0 text-[11px] text-slate-500 dark:text-slate-400">Invitado</span>
  <span v-else-if="provider === 'password'" class="text-[10px] text-slate-500 dark:text-slate-400">Cuenta</span>
</template>
