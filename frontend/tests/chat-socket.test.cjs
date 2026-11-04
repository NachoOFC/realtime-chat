const test = require('node:test')
const assert = require('node:assert/strict')
const { deferred, loadSource } = require('./runtime.cjs')
function setup() {
  const requests = [], connections = [], events = []
  const status = { value: 'idle' }
  class FakeSocket {
    static OPEN = 1
    readyState = 0
    closed = false
    constructor() { connections.push(this) }
    close() { this.closed = true; this.readyState = 3 }
    send() {}
  }
  const { useChatSocket } = loadSource('composables/useChatSocket.ts', {
    useRuntimeConfig: () => ({ public: { wsBase: 'wss://localhost:8080' } }),
    useState: () => status,
    WebSocket: FakeSocket,
  }, name => name === '~/composables/useApi' ? { useApi: () => { const request = deferred(); requests.push(request); return request.promise } } : require(name))
  const client = useChatSocket()
  client.onEvent(event => events.push(event))
  return { client, requests, connections, events, status }
}
test('disconnect cancels a ticket request before it can create a socket', async () => {
  const run = setup()
  const connecting = run.client.connect()
  run.client.disconnect()
  run.requests[0].resolve({ ticket: 'previous-user' })
  await connecting
  assert.equal(run.connections.length, 0)
  assert.equal(run.status.value, 'idle')
})
test('old callbacks and failed ticket requests cannot affect another session', async () => {
  const run = setup()
  const oldRequest = run.client.connect()
  run.client.disconnect()
  const currentRequest = run.client.connect()
  run.requests[1].resolve({ ticket: 'current-user' })
  await currentRequest
  const current = run.connections[0]
  current.readyState = 1
  current.onopen()
  run.requests[0].reject({ statusCode: 401 })
  await oldRequest
  assert.equal(run.status.value, 'open')
  assert.equal(run.events.length, 0)
  run.client.disconnect()
})
test('closed sockets cannot emit events or close a replacement socket', async () => {
  const run = setup()
  const firstRequest = run.client.connect()
  run.requests[0].resolve({ ticket: 'old' })
  await firstRequest
  const old = run.connections[0]
  old.readyState = 1; old.onopen()
  run.client.disconnect()
  const secondRequest = run.client.connect()
  run.requests[1].resolve({ ticket: 'new' })
  await secondRequest
  const current = run.connections[1]
  current.readyState = 1; current.onopen()
  old.onmessage({ data: JSON.stringify({ type: 'user.removed', userId: 'old' }) })
  old.onclose({ code: 4001 })
  old.onerror()
  assert.equal(run.events.length, 0)
  assert.equal(current.closed, false)
  assert.equal(run.status.value, 'open')
  run.client.disconnect()
})
