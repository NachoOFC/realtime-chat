const test = require('node:test')
const assert = require('node:assert/strict')
const { createRequire } = require('node:module')
const { loadSource, deferred } = require('./runtime.cjs')
const pinia = require('pinia')
const vue = createRequire(require.resolve('pinia'))('vue')

function setup() {
  pinia.setActivePinia(pinia.createPinia())
  const values = new Map(), requests = [], routes = []
  const storage = { getItem: key => values.get(key) ?? null, setItem: (key, value) => values.set(key, value), removeItem: key => values.delete(key) }
  const recent = loadSource('utils/recentAccount.ts')
  const { useAuthStore } = loadSource('stores/auth.ts', { window: { localStorage: storage }, navigateTo: async route => routes.push(route) }, name => {
    if (name === 'pinia') return pinia
    if (name === 'vue') return vue
    if (name === '~/utils/recentAccount') return recent
    if (name === '~/stores/chat') return { useChatStore: () => ({ reset() {} }) }
    if (name === '~/composables/useApi') return { useApi: (path, options) => { const request = deferred(); requests.push({ ...request, path, options }); return request.promise } }
    return require(name)
  })
  return { auth: useAuthStore(), values, requests, routes, storage, recent }
}

test('logout remembers only the Google display name and email and still ends the session', async () => {
  const { auth, values, requests, routes, recent, storage } = setup()
  auth.user = { id: 'account-id', username: 'Ana', email: 'ana@example.test', authProvider: 'google', privateValue: 'must-not-be-saved' }
  const signingOut = auth.logout()
  assert.equal(requests[0].path, '/api/auth/logout')
  assert.equal(auth.user.id, 'account-id')
  requests[0].resolve()
  await signingOut
  assert.equal(auth.user, null)
  assert.deepEqual(routes, ['/login'])
  assert.deepEqual({ ...recent.readRecentAccount(storage) }, { username: 'Ana', email: 'ana@example.test' })
  assert.equal([...values.values()].join('').includes('must-not-be-saved'), false)
  assert.equal([...values.values()].join('').includes('account-id'), false)
  auth.forgetRecentAccount()
  assert.equal(recent.readRecentAccount(storage), null)
})

test('guest identities are not remembered as recoverable accounts', () => {
  const { auth, values } = setup()
  auth.user = { id: 'guest-id', username: 'Temporal', authProvider: 'guest', isGuest: true }
  auth.clearSession()
  assert.equal(auth.user, null)
  assert.equal(values.size, 0)
})

test('remembered account information does not authenticate and invalid local data is ignored', () => {
  const { auth, values, recent, storage } = setup()
  recent.saveRecentAccount(storage, { username: 'Ana', email: 'ana@example.test' })
  auth.loadRecentAccount()
  assert.equal(auth.recentAccount.email, 'ana@example.test')
  assert.equal(auth.user, null)
  assert.equal(auth.isAuthenticated, false)
  for (const bad of ['broken json', '{"username":"Ana","email":"invalid"}', 'null']) {
    values.set('chatxor.recentGoogleAccount', bad)
    auth.loadRecentAccount()
    assert.equal(auth.recentAccount, null)
  }
  assert.equal(recent.readRecentAccount({ getItem() { throw Error('blocked') } }), null)
})

test('a failed logout keeps the current account active', async () => {
  const { auth, requests } = setup()
  auth.user = { id: 'account-id', username: 'Ana', email: 'ana@example.test', authProvider: 'google' }
  const signingOut = auth.logout()
  requests[0].reject(Error('unavailable'))
  await assert.rejects(signingOut)
  assert.equal(auth.user.id, 'account-id')
})
