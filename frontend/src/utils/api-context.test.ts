import assert from 'node:assert/strict'
import test from 'node:test'

import { resolveApiUrl, desktopSessionHeader, setDesktopBootstrap, isDefaultEnterpriseRequired } from './api-context.ts'

test('enterprise URLs use the desktop gateway prefix', () => {
  const context = {
    kind: 'enterprise' as const,
    profileId: 'server-a',
    userId: null,
    tenantId: null,
    generation: 7,
  }
  assert.equal(
    resolveApiUrl(context, '/api/v1/agents'),
    '/desktop/remote/profiles/server-a/api/v1/agents',
  )
})

test('personal URLs are unchanged', () => {
  assert.equal(
    resolveApiUrl({ kind: 'personal' as const, generation: 1 }, '/api/v1/agents'),
    '/api/v1/agents',
  )
})

test('desktop bootstrap stores only process session in memory', () => {
  setDesktopBootstrap({ api_base_url: 'http://127.0.0.1:1234', session: 'secret', default_enterprise_server_required: true })
  assert.deepEqual(desktopSessionHeader(), { 'X-Xelora-Desktop-Session': 'secret' })
  assert.equal(isDefaultEnterpriseRequired(), true)
})
