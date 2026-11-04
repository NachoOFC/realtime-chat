import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import type { User } from '~/types'
import { useChatStore } from '~/stores/chat'
import { useApi } from '~/composables/useApi'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const loading = ref(false)
  const initialized = ref(false)

  const isAuthenticated = computed(() => user.value !== null)

  async function ensureSession() {
    if (initialized.value) {
      return
    }

    initialized.value = true

    try {
      user.value = await useApi<User>('/api/me')
    } catch {
      user.value = null
    }
  }

  async function enterAsGuest(username: string) {
    loading.value = true

    try {
      user.value = await useApi<User>('/api/auth/guest', {
        method: 'POST',
        body: { username },
      })
      initialized.value = true
      return user.value
    } finally {
      loading.value = false
    }
  }

  function clearSession() {
    user.value = null
    initialized.value = true
    useChatStore().reset()
  }
  async function logout() {
    await useApi('/api/auth/logout', { method: 'POST' })
    clearSession()
    await navigateTo('/login')
  }

  return {
    user,
    loading,
    initialized,
    isAuthenticated,
    ensureSession,
    enterAsGuest,
    logout,
    clearSession,
  }
})
