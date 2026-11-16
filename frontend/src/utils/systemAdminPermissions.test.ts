import assert from 'node:assert/strict'
import test from 'node:test'

import {
  hasSystemAwareCapability,
  hasSystemAwareTenantRole,
  resolveSystemAwareTenantRole,
} from './systemAdminPermissions.ts'

test('system administrators resolve to Owner without a tenant membership', () => {
  assert.equal(
    resolveSystemAwareTenantRole({
      isSystemAdmin: true,
      canAccessAllTenants: false,
      tenantId: null,
      memberships: [],
    }),
    'owner',
  )
})

test('system administrators satisfy every tenant role gate', () => {
  assert.equal(hasSystemAwareTenantRole('owner', true, ''), true)
  assert.equal(hasSystemAwareTenantRole('owner', false, ''), false)
})

test('ordinary tenant roles keep their existing membership precedence', () => {
  assert.equal(
    resolveSystemAwareTenantRole({
      isSystemAdmin: false,
      canAccessAllTenants: true,
      tenantId: 11,
      memberships: [{ tenant_id: 11, role: 'viewer' }],
    }),
    'viewer',
  )
})

test('system administrators receive every known capability', () => {
  assert.equal(
    hasSystemAwareCapability({
      id: 'can_create_knowledge_base',
      flag: false,
      isSystemAdmin: true,
      isPersonalMode: true,
      isClientProvisioned: true,
    }),
    true,
  )
})

test('unknown capabilities remain denied even for system administrators', () => {
  assert.equal(
    hasSystemAwareCapability({
      id: 'unknown_capability',
      flag: true,
      isSystemAdmin: true,
      isPersonalMode: false,
      isClientProvisioned: false,
    }),
    false,
  )
})
