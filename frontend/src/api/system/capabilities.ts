import { get, post, put } from '@/utils/request'

export interface ClientFeaturesResponse {
  features: {
    workspace_binding?: boolean
  }
}

export async function getClientFeatures(): Promise<ClientFeaturesResponse> {
  return (await get('/api/v1/system/client-features')) as unknown as ClientFeaturesResponse
}

export interface ClientCapabilityFlags {
  can_create_knowledge_base: boolean
  can_create_agent: boolean
  can_bind_model: boolean
}

export interface ClientCapabilityDefaultsResponse {
  defaults: ClientCapabilityFlags
  tighten_existing_client_users: boolean
}

export interface ClientUserRow {
  id: string
  username: string
  email: string
  is_client_provisioned?: boolean
  can_create_knowledge_base?: boolean
  can_create_agent?: boolean
  can_bind_model?: boolean
}

export type AdminUserFilter = 'all' | 'client' | 'non_client' | 'system_admin'

export interface AdminUserRow {
  id: string
  username: string
  email: string
  is_system_admin?: boolean
  is_client_provisioned?: boolean
  can_create_knowledge_base?: boolean
  can_create_agent?: boolean
  can_bind_model?: boolean
  home_tenant_id?: number
  home_tenant_name?: string
}

export async function listAdminUsers(params?: {
  q?: string
  filter?: AdminUserFilter
  offset?: number
  limit?: number
}): Promise<{ users: AdminUserRow[]; total: number }> {
  const qs = new URLSearchParams()
  if (params?.q) qs.set('q', params.q)
  if (params?.filter) qs.set('filter', params.filter)
  if (params?.offset != null) qs.set('offset', String(params.offset))
  if (params?.limit != null) qs.set('limit', String(params.limit))
  const suffix = qs.toString() ? `?${qs.toString()}` : ''
  return (await get(`/api/v1/system/admin/users${suffix}`)) as unknown as {
    users: AdminUserRow[]
    total: number
  }
}

export async function getClientCapabilityDefaults(): Promise<ClientCapabilityDefaultsResponse> {
  return (await get(
    '/api/v1/system/admin/client-capabilities/defaults',
  )) as unknown as ClientCapabilityDefaultsResponse
}

export async function updateClientCapabilityDefaults(
  body: ClientCapabilityFlags & { tighten_existing_client_users: boolean },
): Promise<ClientCapabilityDefaultsResponse> {
  return (await put(
    '/api/v1/system/admin/client-capabilities/defaults',
    body,
  )) as unknown as ClientCapabilityDefaultsResponse
}

export async function tightenExistingClientUsers(): Promise<{ updated_count: number }> {
  return (await post(
    '/api/v1/system/admin/client-capabilities/tighten-existing',
    {},
  )) as unknown as { updated_count: number }
}

export async function listClientUsers(params?: {
  offset?: number
  limit?: number
}): Promise<{ users: ClientUserRow[]; total: number }> {
  const qs = new URLSearchParams()
  if (params?.offset != null) qs.set('offset', String(params.offset))
  if (params?.limit != null) qs.set('limit', String(params.limit))
  const suffix = qs.toString() ? `?${qs.toString()}` : ''
  return (await get(
    `/api/v1/system/admin/client-users${suffix}`,
  )) as unknown as { users: ClientUserRow[]; total: number }
}

export async function updateUserCapabilities(
  userId: string,
  body: ClientCapabilityFlags,
): Promise<ClientUserRow> {
  return (await put(
    `/api/v1/system/admin/users/${userId}/capabilities`,
    body,
  )) as unknown as ClientUserRow
}
