<script setup lang="ts">
import type { Message } from '~/types'

const props = defineProps<{
  message: Message
  own: boolean
}>()

const time = computed(() =>
  new Date(props.message.createdAt).toLocaleTimeString([], {
    hour: '2-digit',
    minute: '2-digit',
  }),
)
</script>

<template>
  <div class="flex" :class="own ? 'justify-end' : 'justify-start'">
    <div
      class="max-w-[75%] rounded-2xl px-3 py-2 text-sm shadow-sm"
      :class="own ? 'bg-slate-900 dark:bg-sky-600 text-white' : 'bg-slate-100 dark:bg-slate-800 text-slate-900 dark:text-slate-100'"
    >
      <p class="whitespace-pre-wrap break-words">{{ message.content }}</p>
      <p class="mt-1 text-right text-[10px]" :class="own ? 'text-slate-300' : 'text-slate-500 dark:text-slate-400'">
        {{ message.status === 'sending' ? 'enviando...' : time }}
      </p>
    </div>
  </div>
</template>
