const test = require('node:test')
const assert = require('node:assert/strict')
const { createRequire } = require('node:module')
const { deferred, loadSource } = require('./runtime.cjs')
const pinia = require('pinia')
const vue = createRequire(require.resolve('pinia'))('vue')
function setup() {
  pinia.setActivePinia(pinia.createPinia())
  const requests = []
  const { useChatStore } = loadSource('stores/chat.ts', {}, name => {
    if (name === 'pinia') return pinia
    if (name === 'vue') return vue
    if (name === '~/composables/useApi') return { useApi: () => { const request = deferred(); requests.push(request); return request.promise } }
    return require(name)
  })
  return { chat: useChatStore(), requests }
}
const message = (id, status = 'sent') => ({ id, status, conversationId: 'chat', senderId: 'user', content: id, createdAt: '2026-11-03T00:00:00Z' })
test('authoritative history removes deleted messages and preserves new arrivals and pending sends', async () => {
  const { chat, requests } = setup()
  chat.upsertMessage(message('deleted'))
  chat.upsertMessage(message('pending', 'sending'))
  const loading = chat.loadMessages('chat')
  chat.upsertMessage(message('arrived-during-fetch'))
  requests[0].resolve({ messages: [message('stored')] })
  await loading
  assert.deepEqual(Array.from(chat.messages.chat, item => item.id), ['stored', 'pending', 'arrived-during-fetch'])
})
test('reconnect invalidates inactive chats and ignores responses from before invalidation', async () => {
  const { chat, requests } = setup()
  chat.upsertMessage(message('deleted-while-offline'))
  const stale = chat.loadMessages('chat')
  chat.invalidateHistories()
  requests[0].resolve({ messages: [message('deleted-while-offline')] })
  await stale
  assert.equal((chat.messages.chat ?? []).length, 0)
  const selecting = chat.selectConversation('chat')
  assert.equal(requests.length, 2)
  requests[1].resolve({ messages: [message('still-stored')] })
  await selecting
  assert.deepEqual(Array.from(chat.messages.chat, item => item.id), ['still-stored'])
})
test('responses from an old identity cannot repopulate a new session', async () => {
  const { chat, requests } = setup()
  const history = chat.loadMessages('chat')
  const conversations = chat.loadConversations()
  const users = chat.loadUsers()
  chat.reset()
  requests[0].resolve({ messages: [message('private-old-user')] })
  requests[1].resolve([{ id: 'old-conversation' }])
  requests[2].resolve([{ id: 'old-contact' }])
  await Promise.all([history, conversations, users])
  assert.equal(Object.keys(chat.messages).length, 0)
  assert.equal(chat.conversations.length, 0)
  assert.equal(chat.users.length, 0)
})
