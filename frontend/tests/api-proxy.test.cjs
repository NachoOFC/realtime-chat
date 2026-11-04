const test = require('node:test')
const assert = require('node:assert/strict')
const http = require('node:http')
const { once } = require('node:events')
const { createRequire } = require('node:module')
const { loadSource } = require('./runtime.cjs')
const h3 = createRequire(require.resolve('nuxt/package.json'))('h3')
const helpers = loadSource('server/utils/apiProxy.ts')

test('proxy uses only a configured HTTPS origin and limits forwarded cookies', () => {
  const request = new URL('https://untrusted.example/api/users?next=https://other.example')
  assert.equal(helpers.buildBackendTarget('https://backend.example', request), 'https://backend.example/api/users?next=https://other.example')
  for (const origin of ['http://backend.example', 'https://user:secret@backend.example', 'https://backend.example/path', 'https://backend.example?query=1', 'https://backend.example#hash']) {
    assert.throws(() => helpers.buildBackendTarget(origin, request))
  }
  assert.throws(() => helpers.buildBackendTarget('https://backend.example', new URL('https://frontend.example/api/../../metadata')))
  assert.throws(() => helpers.buildBackendTarget('http://127.0.0.1:8080', request))
  assert.equal(helpers.buildBackendTarget('http://127.0.0.1:8080', request, true).startsWith('http://127.0.0.1:8080/api/'), true)
  assert.equal(helpers.backendCookies('_ga=tracking; chat_session=token; other=private; chat_google_flow=state'), 'chat_session=token; chat_google_flow=state')
})

async function listen(t, listener) {
  const server = http.createServer(listener)
  server.listen(0, '127.0.0.1')
  await once(server, 'listening')
  t.after(() => new Promise(resolve => { server.closeAllConnections(); server.close(resolve) }))
  return `http://127.0.0.1:${server.address().port}`
}

test('API proxy preserves OAuth redirects and cookies, forwards bodies and enforces no-store', async t => {
  const upstreamRequests = []
  const backend = await listen(t, async (req, res) => {
    let body = ''
    for await (const chunk of req) body += chunk
    upstreamRequests.push({ url: req.url, cookie: req.headers.cookie, origin: req.headers.origin, method: req.method, body })
    res.setHeader('Cache-Control', 'public, max-age=3600')
    if (req.method === 'POST' && req.headers.origin !== 'https://frontend.example') { res.writeHead(403); res.end('forbidden'); return }
    if (req.url === '/api/auth/google') {
      res.setHeader('Set-Cookie', 'chat_google_flow=state; Path=/api/auth/google; Domain=backend.example; HttpOnly; Secure; SameSite=Lax')
      res.writeHead(302, { Location: 'https://accounts.google.com/o/oauth2/v2/auth?state=state' }); res.end(); return
    }
    if (req.url.startsWith('/api/auth/google/callback?')) {
      res.setHeader('Set-Cookie', [
        'chat_google_flow=; Path=/api/auth/google; Domain=backend.example; HttpOnly; Secure; Max-Age=0',
        'chat_session=google; Path=/; Domain=backend.example; HttpOnly; Secure; SameSite=Lax; Expires=Wed, 04 Nov 2026 00:00:00 GMT',
      ])
      res.writeHead(303, { Location: 'https://frontend.example/chat' }); res.end(); return
    }
    res.setHeader('Content-Type', 'application/json')
    if (req.url === '/api/auth/guest') res.setHeader('Set-Cookie', 'chat_session=guest; Path=/; HttpOnly; Secure; SameSite=Lax')
    res.end(JSON.stringify({ ok: true }))
  })
  let target = backend
  const route = loadSource('server/api/[...path].ts', {
    defineEventHandler: h3.defineEventHandler,
    useRuntimeConfig: () => ({ apiProxyTarget: target }),
    createError: h3.createError,
    getRequestURL: h3.getRequestURL,
    getHeader: h3.getHeader,
    proxyRequest: h3.proxyRequest,
    setResponseHeader: h3.setResponseHeader,
  }, name => name === '../utils/apiProxy' ? helpers : require(name)).default
  const app = h3.createApp()
  app.use(route)
  const frontend = await listen(t, h3.toNodeListener(app))
  const start = await fetch(frontend + '/api/auth/google', { redirect: 'manual', headers: { cookie: 'unrelated=secret; chat_session=old' } })
  assert.equal(start.status, 302)
  assert.equal(start.headers.get('location'), 'https://accounts.google.com/o/oauth2/v2/auth?state=state')
  assert.match(start.headers.getSetCookie()[0], /chat_google_flow=state/)
  assert.doesNotMatch(start.headers.getSetCookie()[0], /domain=/i)
  assert.equal(upstreamRequests[0].cookie, 'chat_session=old')
  assert.equal(start.headers.get('cache-control'), 'private, no-store')
  await start.text()
  const callback = await fetch(frontend + '/api/auth/google/callback?code=example&state=state', { redirect: 'manual', headers: { cookie: 'chat_google_flow=state' } })
  assert.equal(callback.status, 303)
  assert.equal(callback.headers.get('location'), 'https://frontend.example/chat')
  assert.equal(callback.headers.getSetCookie().length, 2)
  assert.equal(callback.headers.getSetCookie().some(value => /domain=/i.test(value)), false)
  assert.equal(upstreamRequests[1].cookie, 'chat_google_flow=state')
  assert.equal(upstreamRequests[1].url, '/api/auth/google/callback?code=example&state=state')
  await callback.text()
  const guest = await fetch(frontend + '/api/auth/guest', { method: 'POST', headers: { Origin: 'https://frontend.example', 'Content-Type': 'application/json' }, body: JSON.stringify({ username: 'Invitado' }) })
  assert.equal(guest.status, 200)
  assert.deepEqual(JSON.parse(upstreamRequests[2].body), { username: 'Invitado' })
  assert.equal(upstreamRequests[2].method, 'POST')
  assert.equal(guest.headers.get('netlify-cdn-cache-control'), 'no-store')
  await guest.text()
  const forbidden = await fetch(frontend + '/api/auth/logout', { method: 'POST', headers: { Origin: 'https://other.example' } })
  assert.equal(forbidden.status, 403)
  assert.equal(upstreamRequests[3].origin, 'https://other.example')
  await forbidden.text()
  target = ''
  const missing = await fetch(frontend + '/api/auth/providers')
  assert.equal(missing.status, 503)
  await missing.text()
})
