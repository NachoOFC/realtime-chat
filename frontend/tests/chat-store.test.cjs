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
    if (name === '~/stores/assistant') return { useAssistantStore: () => ({ reset() {} }) }
    if (name === '~/utils/assistant') return { ASSISTANT_CONVERSATION_ID: 'iachat' }
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

test('a removed conversation clears selection and ignores its pending history response', async () => {
  const { chat, requests } = setup()
  chat.conversations = [{ id: 'chat' }]
  const selecting = chat.selectConversation('chat')
  const refreshing = chat.loadConversations()
  chat.nextCursors.chat = 'old-cursor'
  chat.setTyping('chat', 'departed-guest', true)
  requests[1].resolve([])
  await refreshing
  requests[0].resolve({ messages: [message('deleted-guest-message')] })
  await selecting
  assert.equal(chat.activeConversationId, null)
  assert.equal(chat.messages.chat, undefined)
  assert.equal(chat.nextCursors.chat, undefined)
  assert.equal(chat.loadingMessages.chat, undefined)
  assert.equal(chat.typingByConversation.chat, undefined)
  assert.equal(chat.errorMessage, null)
})

test('refreshing direct conversations preserves the assistant selection without loading server history', async () => {
  const { chat, requests } = setup()
  await chat.selectConversation('iachat')
  assert.equal(requests.length, 0)
  const refreshing = chat.loadConversations()
  requests[0].resolve([])
  await refreshing
  assert.equal(chat.activeConversationId, 'iachat')
})

test('a conversation removed before its history loads disappears without an error banner', async () => {
  const { chat, requests } = setup()
  chat.conversations = [{ id: 'chat' }]
  const selecting = chat.selectConversation('chat')
  requests[0].reject({ statusCode: 404 })
  await selecting
  assert.equal(chat.activeConversationId, null)
  assert.equal(chat.conversations.length, 0)
  assert.equal(chat.errorMessage, null)
})

test('creating a group selects it and loads history through the normal conversation API', async () => {
  const { chat, requests } = setup()
  const creating = chat.createGroup('Amigos', ['other'])
  requests[0].resolve({ id: 'group', kind: 'group', name: 'Amigos', peer: { id: '' }, members: [] })
  await Promise.resolve()
  requests[1].resolve({ messages: [] })
  assert.equal(await creating, true)
  assert.equal(chat.activeConversationId, 'group')
  assert.equal(chat.activeConversation.kind, 'group')
})

test('leaving a group invalidates a stale list and drops its cached private history', async () => {
  const { chat, requests } = setup()
  chat.conversations = [{ id: 'chat', kind: 'group' }]
  chat.activeConversationId = 'chat'
  chat.upsertMessage(message('cached'))
  const staleList = chat.loadConversations()
  const leaving = chat.leaveGroup('chat')
  requests[1].resolve()
  assert.equal(await leaving, true)
  requests[0].resolve([{ id: 'chat', kind: 'group' }])
  await staleList
  assert.equal(chat.activeConversationId, null)
  assert.equal(chat.messages.chat, undefined)
  assert.equal(chat.conversations.length, 0)
})

test('a group invitation response from a previous identity is discarded', async () => {
  const { chat, requests } = setup()
  const adding = chat.addGroupMembers('group', ['other'])
  chat.reset()
  requests[0].resolve({ id: 'group', kind: 'group', members: [{ id: 'private-old-member' }] })
  assert.equal(await adding, false)
  assert.equal(chat.conversations.length, 0)
})
