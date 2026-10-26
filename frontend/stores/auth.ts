import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import type { User } from '~/types'
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

  async function login(email: string, password: string) {
    loading.value = true

    try {
      user.value = await useApi<User>('/api/auth/login', {
        method: 'POST',
        body: { email, password },
      })
      initialized.value = true
      return user.value
    } finally {
      loading.value = false
    }
  }

  async function register(email: string, username: string, password: string) {
    loading.value = true

    try {
      user.value = await useApi<User>('/api/auth/register', {
        method: 'POST',
        body: { email, username, password },
      })
      initialized.value = true
      return user.value
    } finally {
      loading.value = false
    }
  }

  async function logout() {
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
    login,
    register,
    logout,
  }
})
