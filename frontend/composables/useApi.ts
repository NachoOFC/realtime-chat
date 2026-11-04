type ApiOptions = {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  body?: Record<string, unknown>
  query?: Record<string, string | number | boolean | undefined>
  headers?: Record<string, string>
}

export function useApi<T>(path: string, options: ApiOptions = {}): Promise<T> {
  const config = useRuntimeConfig()

  // Internal SSR requests forward the current request cookies to the API proxy.
  const requestFetch = import.meta.server && !config.public.apiBase ? useRequestFetch() : $fetch
  return requestFetch<T>(path, {
    baseURL: config.public.apiBase || undefined,
    credentials: 'include',
    method: options.method,
    body: options.body,
    query: options.query,
    headers: { ...(import.meta.server ? useRequestHeaders(['cookie']) : {}), ...options.headers },
  })
}
