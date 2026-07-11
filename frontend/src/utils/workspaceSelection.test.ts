import assert from 'node:assert/strict'
import test from 'node:test'

import { restoreWorkspaceSelection } from './workspaceSelection.ts'

test('restores an available saved workspace', () => {
  const selected = restoreWorkspaceSelection('ws-1', [
    { id: 'ws-1', status: 'available' },
    { id: 'ws-2', status: 'missing' },
  ])

  assert.equal(selected, 'ws-1')
})

test('clears a missing, inaccessible, or unknown saved workspace', () => {
  assert.equal(restoreWorkspaceSelection('ws-1', [{ id: 'ws-1', status: 'missing' }]), '')
  assert.equal(restoreWorkspaceSelection('ws-1', [{ id: 'ws-1', status: 'access_denied' }]), '')
  assert.equal(restoreWorkspaceSelection('unknown', [{ id: 'ws-1', status: 'available' }]), '')
  assert.equal(restoreWorkspaceSelection('', [{ id: 'ws-1', status: 'available' }]), '')
})
