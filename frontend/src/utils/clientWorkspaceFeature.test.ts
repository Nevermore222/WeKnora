import assert from 'node:assert/strict'
import test from 'node:test'

import {
  buildWorkspaceBindingPayload,
  shouldShowWorkspaceBinding,
} from './clientWorkspaceFeature.ts'

test('enterprise workspace binding follows the server feature flag', () => {
  assert.equal(shouldShowWorkspaceBinding('enterprise', false), false)
  assert.equal(shouldShowWorkspaceBinding('enterprise', true), true)
})

test('personal runtime keeps the existing workspace UI', () => {
  assert.equal(shouldShowWorkspaceBinding('personal', false), true)
})

test('enterprise payload uses only a server workspace id', () => {
  assert.deepEqual(
    buildWorkspaceBindingPayload('enterprise', true, 'ws-1'),
    { workspace_binding: { workspace_id: 'ws-1' } },
  )
  assert.deepEqual(buildWorkspaceBindingPayload('enterprise', false, 'ws-1'), {})
})

test('personal payload preserves the existing local root path', () => {
  assert.deepEqual(
    buildWorkspaceBindingPayload('personal', false, 'C:/workspace'),
    { workspace_binding: { root_path: 'C:/workspace' } },
  )
})

test('empty selection never creates a workspace binding', () => {
  assert.deepEqual(buildWorkspaceBindingPayload('enterprise', true, ''), {})
  assert.deepEqual(buildWorkspaceBindingPayload('personal', true, '   '), {})
})
