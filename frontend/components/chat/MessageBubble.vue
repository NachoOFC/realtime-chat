<script setup lang="ts">
import type { Message } from '~/types'

const props = defineProps<{
  message: Message
  own: boolean
  sender?: string
  readLabel?: string
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
      class="max-w-[85%] rounded-lg px-3 py-2 text-sm sm:max-w-[75%]"
      :class="own ? 'bg-slate-900 dark:bg-sky-600 text-white' : 'bg-slate-100 dark:bg-slate-800 text-slate-900 dark:text-slate-100'"
    >
      <p v-if="sender && !own" class="mb-1 text-xs font-medium text-slate-600 dark:text-slate-300">{{ sender }}</p>
      <p class="whitespace-pre-wrap break-words">{{ message.content }}</p>
      <p class="mt-1 text-right text-[10px]" :class="own ? 'text-slate-300' : 'text-slate-500 dark:text-slate-400'">
        {{ message.status === 'sending' ? 'enviando...' : time }}{{ own && message.status === 'read' ? ` · ${readLabel || 'Leído'}` : '' }}
      </p>
    </div>
  </div>
</template>
