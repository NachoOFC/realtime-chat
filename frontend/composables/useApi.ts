type ApiOptions = {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  body?: Record<string, unknown>
  query?: Record<string, string | number | boolean | undefined>
  headers?: Record<string, string>
}

export function useApi<T>(path: string, options: ApiOptions = {}): Promise<T> {
  const config = useRuntimeConfig()

  return $fetch<T>(path, {
    baseURL: config.public.apiBase,
    credentials: 'include',
    method: options.method,
    body: options.body,
    query: options.query,
    headers: options.headers,
  })
}
