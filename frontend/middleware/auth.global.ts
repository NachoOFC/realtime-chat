import { useAuthStore } from '~/stores/auth'
import { useDemoMode } from '~/composables/useDemoMode'

export default defineNuxtRouteMiddleware(async (to) => {
  const demoMode = useDemoMode()
  const auth = useAuthStore()
  await auth.ensureSession()

  if (demoMode.value) {
    if (to.path === '/login') {
      return navigateTo('/chat')
    }
    return
  }

  if (!auth.isAuthenticated && to.path !== '/login') {
    return navigateTo('/login')
  }

  if (auth.isAuthenticated && to.path === '/login') {
    return navigateTo('/chat')
  }
})
