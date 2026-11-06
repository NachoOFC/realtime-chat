<script setup lang="ts">
import { useId } from 'vue'
const props = withDefaults(defineProps<{ open: boolean; title: string; message: string; confirmLabel?: string; busy?: boolean }>(), { confirmLabel: 'Eliminar', busy: false })
const emit = defineEmits<{ cancel: []; confirm: [] }>()
const element = ref<HTMLDialogElement | null>(null)
const titleId = useId()
function sync() { if (props.open && !element.value?.open) element.value?.showModal(); else if (!props.open && element.value?.open) element.value.close() }
onMounted(sync)
watch(() => props.open, sync)
function cancel() { if (!props.busy) emit('cancel') }
</script>
<template>
  <dialog ref="element" :aria-labelledby="titleId" class="w-[calc(100%_-_2rem)] max-w-md rounded-2xl border border-slate-200 bg-white p-6 text-slate-900 shadow-xl backdrop:bg-slate-950/40 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-100" @cancel.prevent="cancel" @click="($event.target === element) && cancel()">
    <div class="mb-4 flex size-11 items-center justify-center rounded-full bg-red-50 text-red-600 dark:bg-red-950 dark:text-red-400"><ChatIcon name="trash" /></div>
    <h2 :id="titleId" class="text-lg font-semibold">{{ title }}</h2>
    <p class="mt-2 text-sm leading-6 text-slate-500 dark:text-slate-400">{{ message }}</p>
    <div class="mt-6 flex justify-end gap-3"><button type="button" :disabled="busy" class="rounded-lg border border-slate-200 px-4 py-2 text-sm hover:bg-slate-50 disabled:opacity-50 dark:border-slate-700 dark:hover:bg-slate-800" @click="cancel">Cancelar</button><button type="button" :disabled="busy" class="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50" @click="emit('confirm')">{{ busy ? 'Eliminando…' : confirmLabel }}</button></div>
  </dialog>
</template>
