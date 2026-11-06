<script setup lang="ts">
import type { Message } from '~/types'

const props = defineProps<{
  message: Message
  own: boolean
  sender?: string
  readLabel?: string
  avatar?: string
  deletable?: boolean
}>()
const emit = defineEmits<{ delete: [message: Message] }>()

const time = computed(() =>
  new Date(props.message.createdAt).toLocaleTimeString([], {
    hour: '2-digit',
    minute: '2-digit',
  }),
)
</script>

<template>
  <div class="group flex items-start gap-3" :class="own ? 'justify-end' : 'justify-start'">
    <ChatAvatar v-if="!own" :name="sender || message.senderName || 'Usuario'" :image="avatar" size="sm" />
    <div class="relative max-w-[83%] sm:max-w-[75%]" :class="own ? 'items-end' : 'items-start'">
      <p v-if="sender && !own" class="mb-1 text-xs font-medium text-slate-600 dark:text-slate-300">{{ sender }}</p>
      <div class="rounded-2xl px-4 py-2.5 text-sm leading-6" :class="own ? 'rounded-tr-sm bg-[#465fff] text-white' : 'rounded-tl-sm bg-[#f2f4f7] text-slate-800 dark:bg-slate-800 dark:text-slate-100'">
        <p class="whitespace-pre-wrap break-words">{{ message.content }}</p>
      </div>
      <div class="mt-1.5 flex items-center gap-2 text-[11px] text-slate-400" :class="own ? 'justify-end' : ''">
        {{ message.status === 'sending' ? 'enviando...' : time }}{{ own && message.status === 'read' ? ` · ${readLabel || 'Leído'}` : '' }}
        <details v-if="own && deletable && message.status !== 'sending'" class="relative">
          <summary class="flex cursor-pointer list-none rounded p-0.5 text-slate-400 hover:bg-slate-100 hover:text-slate-700 focus-visible:outline focus-visible:outline-2 focus-visible:outline-[#465fff] dark:hover:bg-slate-800 dark:hover:text-slate-200 [&::-webkit-details-marker]:hidden" aria-label="Opciones del mensaje"><ChatIcon name="more" :size="15" /></summary>
          <div class="absolute bottom-full right-0 z-20 mb-2 min-w-40 rounded-lg border border-slate-200 bg-white p-1.5 shadow-lg dark:border-slate-700 dark:bg-slate-900"><button type="button" class="flex w-full items-center gap-2 rounded-md px-3 py-2 text-left text-xs text-red-600 hover:bg-red-50 dark:hover:bg-red-950" @click="emit('delete', message); ($event.currentTarget as HTMLElement).closest('details')?.removeAttribute('open')"><ChatIcon name="trash" :size="15" />Eliminar mensaje</button></div>
        </details>
      </div>
    </div>
  </div>
</template>
