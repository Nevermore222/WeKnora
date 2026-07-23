export interface DesktopIdentitySnapshot {
  authenticated: boolean
  user_id?: string
  tenant_id?: number
  user?: Record<string, any>
  tenant?: Record<string, any>
  memberships?: Array<{ tenant_id: number; tenant_name?: string; role: string }>
  expires_at?: string
}

export interface DesktopIdentityState {
  isLoggedIn: boolean
  token: string
  refreshToken: string
  user: {
    id: string
    username: string
    email: string
    avatar?: string
    tenant_id: string
    can_access_all_tenants?: boolean
    is_system_admin?: boolean
    created_at: string
    updated_at: string
  } | null
  tenant: {
    id: string
    name: string
    description?: string
    api_key: string
    status?: string
    business?: string
    owner_id: string
    storage_quota?: number
    storage_used?: number
    created_at: string
    updated_at: string
  } | null
  memberships: Array<{ tenant_id: number; tenant_name?: string; role: string }>
}

export interface CreateDesktopProfilePayload {
  name: string
  base_url: string
  allow_insecure_transport?: boolean
  trusted_certificate_fingerprint?: string
}

export interface DesktopServerProfile {
  id: string
  name: string
  base_url: string
  allow_insecure_transport?: boolean
  trusted_certificate_fingerprint?: string
  created_at?: string
  updated_at?: string
}

export interface DesktopServerCapabilities {
  api_contract_major: number
  api_contract_minor: number
  version?: string
  features?: Record<string, boolean>
}

async function request() {
  return import('@/utils/request')
}

export async function listDesktopProfiles(): Promise<DesktopServerProfile[]> {
  const { get } = await request()
  return get('/desktop/remote/profiles') as unknown as Promise<DesktopServerProfile[]>
}

export async function createDesktopProfile(data: CreateDesktopProfilePayload): Promise<DesktopServerProfile> {
  const { post } = await request()
  return post('/desktop/remote/profiles', data) as unknown as Promise<DesktopServerProfile>
}

export async function activateDesktopProfile(profileId: string): Promise<DesktopServerCapabilities> {
  const { post } = await request()
  return post(`/desktop/remote/profiles/${profileId}/activate`) as unknown as Promise<DesktopServerCapabilities>
}

export async function loginDesktopProfile(profileId: string, data: { email: string; password: string }): Promise<DesktopIdentitySnapshot> {
  const { post } = await request()
  return post(`/desktop/remote/profiles/${profileId}/login`, data) as unknown as Promise<DesktopIdentitySnapshot>
}

export async function getDesktopProfileSession(profileId: string): Promise<DesktopIdentitySnapshot> {
  const { get } = await request()
  return get(`/desktop/remote/profiles/${profileId}/session`) as unknown as Promise<DesktopIdentitySnapshot>
}

export async function logoutDesktopProfile(profileId: string): Promise<void> {
  const { post } = await request()
  return post(`/desktop/remote/profiles/${profileId}/logout`) as unknown as Promise<void>
}

export async function deleteDesktopProfile(profileId: string): Promise<void> {
  const { del } = await request()
  return del(`/desktop/remote/profiles/${profileId}`) as unknown as Promise<void>
}

export async function getDesktopAuthConfig(profileId: string): Promise<{ registration_mode?: string; [key: string]: any }> {
  const { get } = await request()
  return get(`/desktop/remote/profiles/${profileId}/auth/config`) as unknown as Promise<{ registration_mode?: string; [key: string]: any }>
}

export async function registerDesktopProfile(profileId: string, data: { username: string; email: string; password: string }) {
  const { post } = await request()
  return post(`/desktop/remote/profiles/${profileId}/register`, data) as unknown as Promise<{ success: boolean; message?: string; data?: any }>
}

export async function ensureDefaultDesktopProfile(config: {
  name?: string
  base_url: string
  allow_insecure_transport?: boolean
}): Promise<DesktopServerProfile> {
  const normalized = normalizeOrigin(config.base_url)
  const profiles = await listDesktopProfiles()
  const existing = profiles.find((profile) => normalizeOrigin(profile.base_url) === normalized)
  if (existing) {
    await activateDesktopProfile(existing.id)
    return existing
  }
  const created = await createDesktopProfile({
    name: config.name || 'Xelora Server',
    base_url: normalized,
    allow_insecure_transport: config.allow_insecure_transport === true,
  })
  await activateDesktopProfile(created.id)
  return created
}

function normalizeOrigin(value: string): string {
  return String(value || '').trim().replace(/\/+$/, '').toLowerCase()
}

export function applyDesktopIdentitySnapshot(snapshot: DesktopIdentitySnapshot): DesktopIdentityState {
  if (typeof localStorage !== 'undefined') {
    localStorage.removeItem('xelora_token')
    localStorage.removeItem('xelora_refresh_token')
  }

  const tenantID = snapshot.tenant?.id ?? snapshot.tenant_id ?? ''
  const user = snapshot.user
    ? {
        id: String(snapshot.user.id ?? snapshot.user_id ?? ''),
        username: String(snapshot.user.username ?? snapshot.user.email ?? ''),
        email: String(snapshot.user.email ?? ''),
        avatar: snapshot.user.avatar,
        tenant_id: String(snapshot.user.tenant_id ?? tenantID),
        can_access_all_tenants: snapshot.user.can_access_all_tenants === true,
        is_system_admin: snapshot.user.is_system_admin === true,
        created_at: String(snapshot.user.created_at ?? new Date().toISOString()),
        updated_at: String(snapshot.user.updated_at ?? new Date().toISOString()),
      }
    : null
  const tenant = snapshot.tenant
    ? {
        id: String(tenantID),
        name: String(snapshot.tenant.name ?? ''),
        description: snapshot.tenant.description,
        api_key: String(snapshot.tenant.api_key ?? ''),
        status: snapshot.tenant.status,
        business: snapshot.tenant.business,
        owner_id: String(snapshot.tenant.owner_id ?? user?.id ?? ''),
        storage_quota: snapshot.tenant.storage_quota,
        storage_used: snapshot.tenant.storage_used,
        created_at: String(snapshot.tenant.created_at ?? new Date().toISOString()),
        updated_at: String(snapshot.tenant.updated_at ?? new Date().toISOString()),
      }
    : null

  return {
    isLoggedIn: snapshot.authenticated === true && !!user,
    token: '',
    refreshToken: '',
    user,
    tenant,
    memberships: Array.isArray(snapshot.memberships) ? snapshot.memberships : [],
  }
}
