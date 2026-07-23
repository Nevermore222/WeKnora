import assert from 'node:assert/strict'
import test from 'node:test'

import { applyDesktopIdentitySnapshot } from './index.ts'

class MemoryStorage {
  private values = new Map<string, string>()

  get length() {
    return this.values.size
  }

  key(index: number) {
    return Array.from(this.values.keys())[index] ?? null
  }

  getItem(key: string) {
    return this.values.get(key) ?? null
  }

  setItem(key: string, value: string) {
    this.values.set(key, value)
  }

  removeItem(key: string) {
    this.values.delete(key)
  }

  clear() {
    this.values.clear()
  }
}

globalThis.localStorage = new MemoryStorage() as Storage

test('enterprise snapshot authenticates without a token', () => {
  localStorage.setItem('xelora_token', 'old-access')
  localStorage.setItem('xelora_refresh_token', 'old-refresh')

  const state = applyDesktopIdentitySnapshot({
    authenticated: true,
    user: { id: 'u1', username: 'alice', email: 'a@example.com' },
    tenant: { id: 42, name: 'Team' },
    memberships: [{ tenant_id: 42, tenant_name: 'Team', role: 'owner' }],
  })

  assert.equal(state.isLoggedIn, true)
  assert.equal(state.token, '')
  assert.equal(localStorage.getItem('xelora_token'), null)
  assert.equal(localStorage.getItem('xelora_refresh_token'), null)
  assert.equal(state.user?.id, 'u1')
  assert.equal(state.tenant?.id, '42')
  assert.equal(state.memberships[0]?.role, 'owner')
})
