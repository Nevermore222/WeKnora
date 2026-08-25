import assert from 'node:assert/strict'
import test from 'node:test'

import {
  classifyDefaultEnterpriseConnectionError,
  initializeDefaultEnterpriseProfile,
} from './default-enterprise-bootstrap.ts'

test('default enterprise profile initialization retries transient startup failures', async () => {
  const calls: string[] = []
  const contexts: unknown[] = []

  const context = await initializeDefaultEnterpriseProfile(
    {
      default_enterprise_server_url: 'http://localhost:8080',
      default_enterprise_server_name: 'Xelora Server',
      default_enterprise_allow_insecure: true,
    },
    {
      ensureProfile: async (config) => {
        calls.push(config.base_url)
        if (calls.length === 1) {
          throw new Error('backend_not_ready')
        }
        return { id: 'profile-1', name: config.name || '', base_url: config.base_url }
      },
      setContext: (value) => {
        contexts.push(value)
      },
      delay: async () => {},
    },
    { attempts: 2, delayMs: 0 },
  )

  assert.equal(calls.length, 2)
  assert.deepEqual(context, {
    kind: 'enterprise',
    profileId: 'profile-1',
    userId: null,
    tenantId: null,
    generation: 0,
  })
  assert.deepEqual(contexts, [context])
})

test('default enterprise initialization falls back to the local server when configured', async () => {
  const calls: string[] = []

  const context = await initializeDefaultEnterpriseProfile(
    {
      default_enterprise_server_url: 'http://192.168.2.61:8080',
      default_enterprise_server_fallback_url: 'http://127.0.0.1:8080',
    },
    {
      ensureProfile: async (config) => {
        calls.push(config.base_url)
        if (config.base_url === 'http://192.168.2.61:8080') {
          throw new Error('connection timed out')
        }
        return { id: 'profile-local', name: config.name || '', base_url: config.base_url }
      },
      setContext: () => {},
      delay: async () => {},
    },
    { attempts: 1, delayMs: 0 },
  )

  assert.deepEqual(calls, ['http://192.168.2.61:8080', 'http://127.0.0.1:8080'])
  assert.deepEqual(context, {
    kind: 'enterprise',
    profileId: 'profile-local',
    userId: null,
    tenantId: null,
    generation: 0,
  })
})

test('default enterprise initialization preserves the final remote error', async () => {
  await assert.rejects(
    initializeDefaultEnterpriseProfile(
      {
        default_enterprise_server_url: 'http://xelora-server.internal:8080',
      },
      {
        ensureProfile: async () => {
          throw { status: 502, error: 'dial tcp: lookup xelora-server.internal: no such host' }
        },
        setContext: () => {},
        delay: async () => {},
      },
      { attempts: 1, delayMs: 0 },
    ),
    (error: unknown) => {
      assert.ok(error instanceof Error)
      assert.match(error.message, /no such host/)
      return true
    },
  )
})

test('classifies a failed internal server lookup as unreachable', () => {
  assert.equal(
    classifyDefaultEnterpriseConnectionError({
      status: 502,
      error: 'dial tcp: lookup xelora-server.internal: no such host',
    }),
    'unreachable',
  )
})
