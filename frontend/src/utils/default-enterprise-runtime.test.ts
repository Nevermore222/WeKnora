import assert from 'node:assert/strict'
import test from 'node:test'

import {
  createDefaultEnterpriseRuntimeController,
  type DefaultEnterpriseRuntimeControllerDeps,
} from './default-enterprise-runtime.ts'
import type { RuntimeContext } from './api-context.ts'

function createDeps(
  overrides: Partial<DefaultEnterpriseRuntimeControllerDeps> = {},
) {
  let context: RuntimeContext = { kind: 'personal', generation: 0 }
  let initializeCalls = 0
  let switchCalls = 0

  const deps: DefaultEnterpriseRuntimeControllerDeps = {
    getBootstrap: () => ({ default_enterprise_server_url: 'http://server.internal:8080' }),
    getContext: () => context,
    initialize: async () => {
      initializeCalls += 1
      context = {
        kind: 'enterprise',
        profileId: 'profile-1',
        userId: null,
        tenantId: null,
        generation: 0,
      }
      return context
    },
    switchContext: async (next) => {
      switchCalls += 1
      context = { ...next, generation: 1 }
      return context
    },
    ...overrides,
  }

  return {
    deps,
    getContext: () => context,
    getInitializeCalls: () => initializeCalls,
    getSwitchCalls: () => switchCalls,
  }
}

test('default enterprise runtime can be recovered before login', async () => {
  const setup = createDeps()
  const controller = createDefaultEnterpriseRuntimeController(setup.deps)

  const context = await controller.ensure({ switchContext: true })

  assert.equal(context?.kind, 'enterprise')
  assert.equal(setup.getContext().kind, 'enterprise')
  assert.equal(setup.getInitializeCalls(), 1)
  assert.equal(setup.getSwitchCalls(), 1)
})

test('default enterprise runtime does not call recovery without a bootstrap URL', async () => {
  const setup = createDeps({ getBootstrap: () => null })
  const controller = createDefaultEnterpriseRuntimeController(setup.deps)

  assert.equal(await controller.ensure(), null)
  assert.equal(setup.getInitializeCalls(), 0)
})

test('concurrent recovery attempts share one initialization request', async () => {
  let release: (() => void) | undefined
  let initializationRequests = 0
  const setup = createDeps({
    initialize: () => {
      initializationRequests += 1
      const promise = new Promise<RuntimeContext>((resolve) => {
        release = () => resolve({
          kind: 'enterprise',
          profileId: 'profile-1',
          userId: null,
          tenantId: null,
          generation: 0,
        })
      })
      return promise
    },
  })
  const controller = createDefaultEnterpriseRuntimeController(setup.deps)

  const first = controller.ensure()
  const second = controller.ensure()
  release?.()

  await Promise.all([first, second])
  assert.equal(initializationRequests, 1)
})
