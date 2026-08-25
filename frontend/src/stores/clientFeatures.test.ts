import assert from 'node:assert/strict'
import test from 'node:test'

import { createClientFeaturesStoreForTest } from './clientFeatures.ts'

test('workspace binding is disabled until enterprise server enables it', async () => {
  const store = createClientFeaturesStoreForTest({
    kind: 'enterprise',
    fetchFeatures: async () => ({ features: { workspace_binding: false } }),
  })
  await store.refresh()
  assert.equal(store.workspaceBindingEnabled, false)
})

test('refresh enables workspace binding from the server response', async () => {
  const store = createClientFeaturesStoreForTest({
    kind: 'enterprise',
    fetchFeatures: async () => ({ features: { workspace_binding: true } }),
  })
  await store.refresh()
  assert.equal(store.workspaceBindingEnabled, true)
})

test('refresh failure hides only enterprise workspace UI', async () => {
  const store = createClientFeaturesStoreForTest({
    kind: 'enterprise',
    fetchFeatures: async () => {
      throw new Error('server_unavailable')
    },
  })
  await store.refresh()
  assert.equal(store.workspaceBindingEnabled, false)
  assert.equal(store.loaded, true)
})

test('personal runtime does not request the enterprise feature', async () => {
  let calls = 0
  const store = createClientFeaturesStoreForTest({
    kind: 'personal',
    fetchFeatures: async () => {
      calls += 1
      return { features: { workspace_binding: true } }
    },
  })
  await store.refresh()
  assert.equal(calls, 0)
  assert.equal(store.loaded, true)
  assert.equal(store.workspaceBindingEnabled, false)
})

test('reset clears a previously enabled feature', async () => {
  const store = createClientFeaturesStoreForTest({
    kind: 'enterprise',
    fetchFeatures: async () => ({ features: { workspace_binding: true } }),
  })
  await store.refresh()
  store.reset()
  assert.equal(store.loaded, false)
  assert.equal(store.workspaceBindingEnabled, false)
})
