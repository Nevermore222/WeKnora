import assert from 'node:assert/strict'
import test from 'node:test'

import {
  resolveAgentModeEnabled,
  resolveContinuableAssistantMessageId,
  resolveStreamApiBaseUrl,
} from './chat-runtime.ts'

test('shared smart-reasoning agent enables the agent pipeline', () => {
  assert.equal(
    resolveAgentModeEnabled('custom-agent', 'smart-reasoning', false),
    true,
  )
})

test('quick-answer agent disables the agent pipeline', () => {
  assert.equal(
    resolveAgentModeEnabled('custom-agent', 'quick-answer', true),
    false,
  )
})

test('incomplete assistant without an id cannot start continue-stream', () => {
  assert.equal(
    resolveContinuableAssistantMessageId({
      role: 'assistant',
      is_completed: false,
    }),
    null,
  )
})

test('incomplete assistant with an id can start continue-stream', () => {
  assert.equal(
    resolveContinuableAssistantMessageId({
      id: 'message-1',
      role: 'assistant',
      is_completed: false,
    }),
    'message-1',
  )
})

test('desktop streaming prefers the direct embedded backend URL', () => {
  assert.equal(
    resolveStreamApiBaseUrl('http://127.0.0.1:54321', ''),
    'http://127.0.0.1:54321',
  )
})

test('web streaming keeps the configured web base URL', () => {
  assert.equal(resolveStreamApiBaseUrl('', '/xelora'), '/xelora')
})
