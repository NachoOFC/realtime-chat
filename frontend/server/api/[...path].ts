import { buildBackendTarget, backendCookies } from '../utils/apiProxy'

export default defineEventHandler(async event => {
  const config = useRuntimeConfig(event)
  if (!config.apiProxyTarget) {
    throw createError({ statusCode: 503, statusMessage: 'El backend no esta configurado' })
  }
  let target: string
  try {
    target = buildBackendTarget(config.apiProxyTarget, getRequestURL(event), import.meta.dev)
  } catch {
    throw createError({ statusCode: 503, statusMessage: 'La URL del backend no es valida' })
  }
  return proxyRequest(event, target, {
    // Google redirects must reach the browser with their state cookie intact.
    fetchOptions: { redirect: 'manual' },
    headers: { cookie: backendCookies(getHeader(event, 'cookie')) },
    cookieDomainRewrite: { '*': '' },
    onResponse(proxyEvent) {
      setResponseHeader(proxyEvent, 'cache-control', 'private, no-store')
      setResponseHeader(proxyEvent, 'cdn-cache-control', 'no-store')
      setResponseHeader(proxyEvent, 'netlify-cdn-cache-control', 'no-store')
    },
  })
})
