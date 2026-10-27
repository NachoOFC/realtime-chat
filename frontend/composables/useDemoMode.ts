import { computed } from 'vue'
import { isDemoMode } from '~/utils/demo'

export function useDemoMode() {
  const config = useRuntimeConfig()
  return computed(() => isDemoMode(config.public.demoMode))
}
