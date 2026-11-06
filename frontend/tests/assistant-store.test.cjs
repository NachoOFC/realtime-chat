const test = require('node:test')
const assert = require('node:assert/strict')
const { createRequire } = require('node:module')
const { webcrypto } = require('node:crypto')
const { deferred, loadSource } = require('./runtime.cjs')
const pinia = require('pinia')
const vue = createRequire(require.resolve('pinia'))('vue')
const helpers = loadSource('utils/assistant.ts')

function setup() {
  pinia.setActivePinia(pinia.createPinia())
  const requests = []
  const { useAssistantStore } = loadSource('stores/assistant.ts', { AbortController, crypto: webcrypto }, name => {
    if (name === 'pinia') return pinia
    if (name === 'vue') return vue
    if (name === '~/utils/assistant') return helpers
    if (name === '~/composables/useApi') return { useApi: (path, options) => {
      const request = { ...deferred(), path, options }
      requests.push(request)
      return request.promise
    } }
    return require(name)
  })
  return { assistant: useAssistantStore(), requests }
}

let nextMessage = 0
const message = (senderId, content, conversationId = 'iachat', status = 'sent') => ({
  id: `message-${++nextMessage}`,
  conversationId,
  senderId,
  content,
  status,
  createdAt: '2026-11-04T00:00:00Z',
})
const status = enabled => ({ enabled, provider: 'gemini', model: 'test-model' })

test('availability handles enabled, disabled and failed checks and allows refreshing', async () => {
  const { assistant, requests } = setup()
  assert.equal(assistant.enabled, false)
  assert.equal(assistant.statusLoaded, false)
  const first = assistant.loadStatus()
  await assistant.loadStatus()
  assert.equal(requests.length, 1)
  assert.equal(requests[0].path, '/api/assistant')
  assert.equal(assistant.loadingStatus, true)
  requests[0].resolve(status(true))
  await first
  assert.equal(assistant.enabled, true)
  assert.equal(assistant.statusLoaded, true)
  assert.equal(assistant.loadingStatus, false)

  const second = assistant.loadStatus()
  requests[1].resolve(status(false))
  await second
  assert.equal(assistant.enabled, false)
  assert.equal(assistant.errorMessage, null)

  const third = assistant.loadStatus()
  requests[2].reject(new Error('internal provider details'))
  await third
  assert.equal(assistant.enabled, false)
  assert.equal(assistant.loadingStatus, false)
  assert.match(assistant.errorMessage, /Actualizar|Actualiza/)
  assert.doesNotMatch(assistant.errorMessage, /internal provider details/)
})

test('sending uses only this assistant chat and only complete pairs from the current identity', async () => {
  const { assistant, requests } = setup()
  assistant.enabled = true
  assistant.messages = [
    message('user', 'private message from another chat', 'other-chat'),
    message('old-identity', 'old private session'),
    message('assistant', 'old reply'),
    message('user', 'first question'),
    message('assistant', 'first answer'),
    message('user', 'unconfirmed message', 'iachat', 'sending'),
  ]
  const sending = assistant.send('  second question  ', 'user')
  assert.equal(requests[0].path, '/api/assistant/messages')
  assert.equal(requests[0].options.method, 'POST')
  assert.deepEqual(Array.from(requests[0].options.body.messages, turn => ({ ...turn })), [
    { role: 'user', content: 'first question' },
    { role: 'assistant', content: 'first answer' },
    { role: 'user', content: 'second question' },
  ])
  assert.equal(assistant.sending, true)
  assert.equal(assistant.messages.at(-1).status, 'sending')
  assert.equal(await assistant.send('another simultaneous question', 'user'), false)
  assert.equal(requests.length, 1)

  requests[0].resolve({ content: '  actual provider answer  ' })
  assert.equal(await sending, true)
  assert.equal(assistant.sending, false)
  assert.equal(assistant.messages.at(-2).status, 'sent')
  assert.equal(assistant.messages.at(-1).senderId, 'assistant')
  assert.equal(assistant.messages.at(-1).conversationId, 'iachat')
  assert.equal(assistant.messages.at(-1).content, 'actual provider answer')
})

test('history keeps the latest ten complete pairs and always starts and ends with a user', () => {
  const messages = []
  for (let index = 0; index < 12; index++) {
    messages.push(message('user', `question-${index}`), message('assistant', `answer-${index}`))
  }
  const history = helpers.buildAssistantHistory(messages, 'user', 'latest question')
  assert.equal(history.length, 21)
  assert.equal(history[0].content, 'question-2')
  assert.equal(history.at(-1).content, 'latest question')
  history.forEach((turn, index) => assert.equal(turn.role, index % 2 === 0 ? 'user' : 'assistant'))
})

test('history trims whole pairs at the rune limit and counts emoji as single characters', () => {
  const messages = []
  for (let index = 0; index < 9; index++) {
    messages.push(message('user', `question-${index}`), message('assistant', '😀'.repeat(2000)))
  }
  const history = helpers.buildAssistantHistory(messages, 'user', 'latest')
  assert.equal(history.length, 15)
  assert.equal(history[0].content, 'question-2')
  const total = history.reduce((sum, turn) => sum + Array.from(turn.content).length, 0)
  assert.ok(total <= 16000)
  assert.equal(history.at(-1).role, 'user')
})

test('failed sends roll back the pending message and can be retried without duplicate context', async () => {
  const { assistant, requests } = setup()
  assistant.enabled = true
  assistant.messages = [message('user', 'prior question'), message('assistant', 'prior answer')]
  const failed = assistant.send('retry this', 'user')
  requests[0].reject({ statusCode: 429, message: 'sensitive provider response' })
  assert.equal(await failed, false)
  assert.equal(assistant.sending, false)
  assert.equal(assistant.messages.length, 2)
  assert.equal(assistant.messages.some(item => item.status === 'sending'), false)
  assert.match(assistant.errorMessage, /límite de uso/)
  assert.doesNotMatch(assistant.errorMessage, /sensitive provider response/)

  const retry = assistant.send('retry this', 'user')
  assert.equal(requests[1].options.body.messages.length, 3)
  assert.equal(assistant.errorMessage, null)
  requests[1].resolve({ content: 'recovered answer' })
  assert.equal(await retry, true)
  assert.equal(assistant.messages.length, 4)
})

test('empty provider responses fail without creating fake replies or stranded pending messages', async () => {
  const { assistant, requests } = setup()
  assistant.enabled = true
  const sending = assistant.send('question', 'user')
  requests[0].resolve({ content: '  ' })
  assert.equal(await sending, false)
  assert.equal(assistant.messages.length, 0)
  assert.equal(assistant.sending, false)
  assert.ok(assistant.errorMessage)
})

test('logout cancels availability checks and ignores their response in a new session', async () => {
  const { assistant, requests } = setup()
  const oldStatus = assistant.loadStatus()
  assistant.reset()
  assert.equal(requests[0].options.signal.aborted, true)
  assert.equal(assistant.statusLoaded, false)
  assert.equal(assistant.loadingStatus, false)
  const newStatus = assistant.loadStatus()
  requests[0].resolve(status(true))
  await oldStatus
  assert.equal(assistant.enabled, false)
  assert.equal(assistant.loadingStatus, true)
  requests[1].resolve(status(false))
  await newStatus
  assert.equal(assistant.enabled, false)
  assert.equal(assistant.loadingStatus, false)
})

test('logout cancels generation and prevents late replies from entering the next identity', async () => {
  const { assistant, requests } = setup()
  assistant.enabled = true
  const oldSend = assistant.send('private old question', 'old-user')
  assistant.reset()
  assert.equal(requests[0].options.signal.aborted, true)
  assert.equal(assistant.messages.length, 0)
  assert.equal(assistant.errorMessage, null)
  assert.equal(assistant.sending, false)

  assistant.enabled = true
  const newSend = assistant.send('new question', 'new-user')
  requests[0].resolve({ content: 'private old answer' })
  assert.equal(await oldSend, false)
  assert.equal(assistant.sending, true)
  assert.equal(assistant.messages.length, 1)
  assert.equal(assistant.messages[0].senderId, 'new-user')
  assert.equal(requests[1].options.body.messages.length, 1)
  requests[1].resolve({ content: 'new answer' })
  assert.equal(await newSend, true)
  assert.deepEqual(Array.from(assistant.messages, item => item.content), ['new question', 'new answer'])
})

test('disabled, anonymous, empty and oversized sends never reach the API', async () => {
  const { assistant, requests } = setup()
  assert.equal(await assistant.send('question', 'user'), false)
  assistant.enabled = true
  assert.equal(await assistant.send('question', ''), false)
  assert.equal(await assistant.send('   ', 'user'), false)
  assert.equal(await assistant.send('a'.repeat(4001), 'user'), false)
  assert.equal(requests.length, 0)

  const validEmoji = assistant.send('😀'.repeat(4000), 'user')
  assert.equal(requests.length, 1)
  requests[0].resolve({ content: 'answer' })
  assert.equal(await validEmoji, true)
})
