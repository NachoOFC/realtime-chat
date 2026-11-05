import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import type { User } from '~/types'
import { useChatStore } from '~/stores/chat'
import { useApi } from '~/composables/useApi'
import { readRecentAccount, saveRecentAccount, removeRecentAccount, type RecentAccount } from '~/utils/recentAccount'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const loading = ref(false)
  const initialized = ref(false)
  const recentAccount = ref<RecentAccount | null>(null)

  function loadRecentAccount() {
    if (!import.meta.client) return
    try { recentAccount.value = readRecentAccount(window.localStorage) } catch { recentAccount.value = null }
  }
  function rememberCurrentAccount() {
    if (!import.meta.client || user.value?.authProvider !== 'google' || !user.value.email) return
    const account = { username: user.value.username, email: user.value.email }
    try { if (saveRecentAccount(window.localStorage, account)) recentAccount.value = account } catch { /* Storage may be blocked. */ }
  }
  function forgetRecentAccount() {
    recentAccount.value = null
    if (import.meta.client) { try { removeRecentAccount(window.localStorage) } catch { /* Storage may be blocked. */ } }
  }

  const isAuthenticated = computed(() => user.value !== null)

  async function ensureSession() {
    if (initialized.value) {
      return
    }

    initialized.value = true

    try {
      user.value = await useApi<User>('/api/me')
      rememberCurrentAccount()
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
    rememberCurrentAccount()
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
    recentAccount,
    loadRecentAccount,
    rememberCurrentAccount,
    forgetRecentAccount,
    ensureSession,
    enterAsGuest,
    logout,
    clearSession,
  }
})
