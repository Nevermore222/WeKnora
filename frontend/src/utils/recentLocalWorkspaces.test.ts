import { beforeEach, describe, expect, it } from 'vitest'
import {
  dismissRecentLocalWorkspace,
  loadRecentLocalWorkspaces,
  rememberRecentLocalWorkspace,
} from './recentLocalWorkspaces'

const store = new Map<string, string>()

const localStorageMock = {
  getItem: (key: string) => (store.has(key) ? store.get(key)! : null),
  setItem: (key: string, value: string) => {
    store.set(key, value)
  },
  removeItem: (key: string) => {
    store.delete(key)
  },
  clear: () => {
    store.clear()
  },
}

Object.defineProperty(globalThis, 'localStorage', {
  value: localStorageMock,
  configurable: true,
})

describe('recentLocalWorkspaces', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('remembers and orders by last used', () => {
    rememberRecentLocalWorkspace({
      workspace_id: 'local:a',
      workspace_name: 'a',
      root_path: 'D:\\a',
    })
    rememberRecentLocalWorkspace({
      workspace_id: 'local:b',
      workspace_name: 'b',
      root_path: 'D:\\b',
    })
    const list = loadRecentLocalWorkspaces()
    expect(list[0].root_path).toBe('D:\\b')
    expect(list).toHaveLength(2)
  })

  it('dismisses by path', () => {
    rememberRecentLocalWorkspace({
      workspace_id: 'local:a',
      workspace_name: 'a',
      root_path: 'D:\\a',
    })
    dismissRecentLocalWorkspace('D:\\a')
    expect(loadRecentLocalWorkspaces()).toHaveLength(0)
  })
})
