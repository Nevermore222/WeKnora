import assert from 'node:assert/strict'
import test from 'node:test'

import { initializeDefaultEnterpriseProfile } from './default-enterprise-bootstrap.ts'

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
