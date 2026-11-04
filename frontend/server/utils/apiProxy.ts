/** Build a fixed-origin API target; incoming URLs never select the backend host. */
export function buildBackendTarget(configuredOrigin: string, incoming: URL, allowLocalHttp = false): string {
  const upstream = new URL(configuredOrigin)
  const local = upstream.hostname === 'localhost' || upstream.hostname === '127.0.0.1' || upstream.hostname === '[::1]'
  if ((upstream.protocol !== 'https:' && !(allowLocalHttp && local && upstream.protocol === 'http:')) || upstream.username || upstream.password || upstream.search || upstream.hash || upstream.pathname !== '/') {
    throw new Error('NUXT_API_PROXY_TARGET must be an HTTPS origin')
  }
  if (!incoming.pathname.startsWith('/api/')) throw new Error('Only API paths can be proxied')
  upstream.pathname = incoming.pathname
  upstream.search = incoming.search
  return upstream.href
}

/** Do not send unrelated Netlify or analytics cookies to the backend. */
export function backendCookies(raw: string | undefined): string {
  return (raw ?? '').split(';').map(cookie => cookie.trim()).filter(cookie => /^(chat_session|chat_google_flow)=/.test(cookie)).join('; ')
}
