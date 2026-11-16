export type TenantRole = 'viewer' | 'contributor' | 'admin' | 'owner'

export interface TenantRoleResolutionInput {
  isSystemAdmin: boolean
  canAccessAllTenants: boolean
  tenantId: string | number | null | undefined
  memberships: ReadonlyArray<{ tenant_id: number; role: string }>
}

export interface CapabilityResolutionInput {
  id: string
  flag: boolean | undefined
  isSystemAdmin: boolean
  isPersonalMode: boolean
  isClientProvisioned: boolean
}

const ROLE_LEVEL: Record<TenantRole, number> = {
  viewer: 10,
  contributor: 20,
  admin: 30,
  owner: 40,
}

const KNOWN_CAPABILITIES = new Set([
  'can_create_knowledge_base',
  'can_create_agent',
  'can_bind_model',
])

function isTenantRole(role: string): role is TenantRole {
  return role in ROLE_LEVEL
}

export function resolveSystemAwareTenantRole(input: TenantRoleResolutionInput): TenantRole | '' {
  if (input.isSystemAdmin) return 'owner'

  const tenantId =
    input.tenantId === null || input.tenantId === undefined
      ? ''
      : String(input.tenantId)
  if (!tenantId) return ''

  const membership = input.memberships.find(
    (item) => String(item.tenant_id) === tenantId,
  )
  if (membership && isTenantRole(membership.role)) return membership.role

  // Cross-tenant superusers receive a temporary Admin role on the server.
  if (input.canAccessAllTenants) return 'admin'
  return ''
}

export function hasSystemAwareTenantRole(
  minimum: TenantRole,
  isSystemAdmin: boolean,
  role: TenantRole | '',
): boolean {
  if (isSystemAdmin) return true
  return (role ? ROLE_LEVEL[role] : 0) >= ROLE_LEVEL[minimum]
}

export function hasSystemAwareCapability(input: CapabilityResolutionInput): boolean {
  if (!KNOWN_CAPABILITIES.has(input.id)) return false
  if (input.isSystemAdmin) return true
  if (input.flag === false) return false
  if (input.isPersonalMode) {
    return input.isClientProvisioned && input.flag === true
  }
  if (input.isClientProvisioned) return input.flag === true
  return true
}
