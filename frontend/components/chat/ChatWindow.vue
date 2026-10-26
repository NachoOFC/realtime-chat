<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'
import { useChatStore } from '~/stores/chat'

const chat = useChatStore()
const auth = useAuthStore()

const scroller = ref<HTMLElement | null>(null)

const peerTyping = computed(() =>
  chat.activeTypingUsers.some((userId: string) => userId !== auth.user?.id),
)

async function scrollToBottom() {
  await nextTick()
  const element = scroller.value
  if (element) {
    element.scrollTop = element.scrollHeight
  }
}

watch(() => chat.activeConversationId, scrollToBottom)
watch(() => chat.activeMessages.length, scrollToBottom)
</script>

<template>
  <section
    v-if="chat.activeConversation"
    class="flex min-w-0 flex-1 flex-col"
  >
    <header class="flex items-center gap-3 border-b border-slate-200 px-4 py-3">
      <button
        type="button"
        class="rounded-lg border border-slate-300 px-2 py-1 text-sm text-slate-600 md:hidden"
        @click="chat.activeConversationId = null"
      >
        Volver
      </button>

      <span
        class="flex size-9 items-center justify-center rounded-full bg-slate-200 text-sm font-semibold text-slate-700"
      >
        {{ chat.activeConversation.peer.username.slice(0, 1).toUpperCase() }}
      </span>

      <div>
        <p class="font-medium">{{ chat.activeConversation.peer.username }}</p>
        <p class="text-xs text-slate-500">Chat 1:1</p>
      </div>
    </header>

    <div ref="scroller" class="flex-1 space-y-2 overflow-y-auto bg-white px-4 py-4">
      <p v-if="!chat.activeMessages.length" class="py-10 text-center text-sm text-slate-500">
        Todavía no hay mensajes en esta conversación.
      </p>

      <MessageBubble
        v-for="message in chat.activeMessages"
        :key="message.id"
        :message="message"
        :own="message.senderId === auth.user?.id"
      />
    </div>

    <TypingIndicator :active="peerTyping" />

    <MessageComposer :conversation-id="chat.activeConversation.id" />
  </section>

  <section v-else class="flex flex-1 items-center justify-center bg-white max-md:hidden">
    <p class="text-sm text-slate-500">Selecciona una conversación para empezar.</p>
  </section>
</template>
