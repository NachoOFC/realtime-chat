export interface RecentAccount {
  username: string
  email: string
}

const key = 'chatxor.recentGoogleAccount'
type AccountStorage = Pick<Storage, 'getItem' | 'setItem' | 'removeItem'>

function validAccount(value: unknown): value is RecentAccount {
  if (!value || typeof value !== 'object') return false
  const account = value as RecentAccount
  return typeof account.username === 'string' && account.username.length > 0 && account.username.length <= 80
    && typeof account.email === 'string' && account.email.length <= 254 && /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(account.email)
}

export function readRecentAccount(storage: AccountStorage): RecentAccount | null {
  try {
    const value: unknown = JSON.parse(storage.getItem(key) || 'null')
    return validAccount(value) ? { username: value.username, email: value.email } : null
  } catch { return null }
}

export function saveRecentAccount(storage: AccountStorage, account: RecentAccount): boolean {
  if (!validAccount(account)) return false
  try { storage.setItem(key, JSON.stringify({ username: account.username, email: account.email })); return true }
  catch { return false }
}

export function removeRecentAccount(storage: AccountStorage) {
  try { storage.removeItem(key) } catch { /* Account storage is optional. */ }
}
