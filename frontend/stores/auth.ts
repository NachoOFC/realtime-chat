import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import type { User } from '~/types'
import { useApi } from '~/composables/useApi'
import { useDemoMode } from '~/composables/useDemoMode'
import { demoUser } from '~/utils/demo'

export const useAuthStore = defineStore('auth', () => {
  const demoMode = useDemoMode()
  const user = ref<User | null>(null)
  const loading = ref(false)
  const initialized = ref(false)

  const isAuthenticated = computed(() => user.value !== null)

  async function ensureSession() {
    if (initialized.value) {
      return
    }

    initialized.value = true

    if (demoMode.value) {
      user.value = { ...demoUser }
      return
    }

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

  async function logout() {
    if (demoMode.value) {
      user.value = { ...demoUser }
      return
    }

    try {
      await useApi('/api/auth/logout', { method: 'POST' })
    } finally {
      user.value = null
      initialized.value = true
      await navigateTo('/login')
    }
  }

  return {
    user,
    loading,
    initialized,
    isAuthenticated,
    ensureSession,
    enterAsGuest,
    logout,
  }
})
