<script setup lang="ts">
import { useChatStore } from '~/stores/chat'

const chat = useChatStore()
</script>

<template>
  <aside
    class="flex w-80 shrink-0 flex-col border-r border-slate-200 dark:border-slate-700 bg-slate-50/60 dark:bg-slate-950/60 max-md:w-full"
    :class="{ 'max-md:hidden': chat.activeConversationId }"
  >
    <div class="border-b border-slate-200 dark:border-slate-700 px-4 py-3">
      <h2 class="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
        Conversaciones
      </h2>
    </div>

    <div v-if="chat.loadingConversations" class="p-4 text-sm text-slate-500 dark:text-slate-400">
      Cargando...
    </div>

    <ul v-else-if="chat.conversations.length" class="flex-1 overflow-y-auto">
      <li v-for="conversation in chat.conversations" :key="conversation.id">
        <button
          type="button"
          class="flex w-full items-center gap-3 px-4 py-3 text-left transition hover:bg-white dark:hover:bg-slate-800"
          :class="{ 'bg-white dark:bg-slate-900': chat.activeConversationId === conversation.id }"
          @click="chat.selectConversation(conversation.id)"
        >
          <span
            class="flex size-10 shrink-0 items-center justify-center rounded-full bg-slate-200 dark:bg-slate-700 text-sm font-semibold text-slate-700 dark:text-slate-200"
          >
            {{ conversation.peer.username.slice(0, 1).toUpperCase() }}
          </span>

          <span class="min-w-0 flex-1">
            <span class="flex items-center justify-between gap-2">
              <span class="truncate font-medium">{{ conversation.peer.username }}</span>
              <PresenceDot :online="false" />
            </span>
            <span class="mt-0.5 block truncate text-xs text-slate-500 dark:text-slate-400">
              {{ conversation.lastMessage?.content ?? 'Sin mensajes' }}
            </span>
          </span>
        </button>
      </li>
    </ul>

    <p v-else class="p-4 text-sm text-slate-500 dark:text-slate-400">
      Aún no hay conversaciones. Crea una desde `POST /api/conversations`.
    </p>
  </aside>
</template>
