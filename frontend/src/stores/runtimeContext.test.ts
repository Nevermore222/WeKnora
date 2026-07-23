import assert from 'node:assert/strict'
import test from 'node:test'

import {
  acceptResponse,
  getRuntimeContext,
  switchRuntimeContext,
  type RuntimeContext,
} from '../utils/api-context.ts'
import {
  registerContextAborter,
  registerContextReset,
  resetContextRegistriesForTest,
} from '../utils/context-reset.ts'

test('switchRuntimeContext aborts requests, resets stores, then activates destination', async () => {
  resetContextRegistriesForTest()
  const events: string[] = []
  const controller = new AbortController()
  controller.signal.addEventListener('abort', () => events.push('abort'))
  registerContextAborter(controller)
  registerContextReset(async () => {
    events.push(`reset:${getRuntimeContext().kind}`)
  })

  const destination: RuntimeContext = {
    kind: 'enterprise',
    profileId: 'server-a',
    userId: 'u1',
    tenantId: 42,
    generation: 0,
  }

  const next = await switchRuntimeContext(destination)

  assert.deepEqual(events, ['abort', 'reset:personal'])
  assert.equal(next.kind, 'enterprise')
  assert.equal(next.generation, 1)
  assert.equal(getRuntimeContext().kind, 'enterprise')
})

test('acceptResponse rejects prior context generation', () => {
  assert.equal(acceptResponse({ generation: 3 }, 4), false)
  assert.equal(acceptResponse({ generation: 4 }, 4), true)
})
