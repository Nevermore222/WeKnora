import { get, post, put, del } from '@/utils/request'

export interface EnterpriseServer {
  id: string
  name: string
  base_url: string
  auto_connect: boolean
  created_at: string
  updated_at: string
  last_seen_at?: string
  status?: string
  last_error?: string
  linked_email?: string
}

export interface CreateServerPayload {
  name: string
  base_url: string
  api_token?: string
  auto_connect?: boolean
}

export interface UpdateServerPayload {
  name?: string
  base_url?: string
  api_token?: string
  auto_connect?: boolean
}

export interface AggregatedResource {
  id: string
  type: 'knowledge_base' | 'agent' | 'skill' | 'model'
  name: string
  description?: string
  origin: 'local' | 'enterprise'
  server_id?: string
  server_name?: string
  available: boolean
  shared?: boolean
  permission?: string
  org_name?: string
}

// --- Server CRUD ---

export function listServers() {
  return get('/api/v1/enterprise/servers')
}

export function createServer(data: CreateServerPayload) {
  return post('/api/v1/enterprise/servers', data)
}

export function updateServer(id: string, data: UpdateServerPayload) {
  return put(`/api/v1/enterprise/servers/${id}`, data)
}

export function deleteServer(id: string) {
  return del(`/api/v1/enterprise/servers/${id}`)
}

export function testServer(id: string) {
  return post(`/api/v1/enterprise/servers/${id}/test`)
}

export function connectServer(id: string) {
  return post(`/api/v1/enterprise/servers/${id}/connect`)
}

export function disconnectServer(id: string) {
  return post(`/api/v1/enterprise/servers/${id}/disconnect`)
}

export function getServerStatus(id: string) {
  return get(`/api/v1/enterprise/servers/${id}/status`)
}

// --- Discovery ---

export function discoverServers() {
  return get('/api/v1/enterprise/discover')
}

// --- Resources ---

export function listResources(type?: string) {
  const params = type ? { type } : {}
  return get('/api/v1/enterprise/resources', { params })
}

// --- Proxy ---

// Create a session on the enterprise server (for enterprise agent chats, which
// keep their multi-turn context server-side). Returns the server's response
// including the new session id at data.id.
export function createServerSession(serverId: string, title: string) {
  return post('/api/v1/enterprise/sessions', { title }, {
    headers: { 'X-Enterprise-Server-ID': serverId },
  })
}

export function proxyChat(serverId: string, sessionId: string, payload: Record<string, unknown>) {
  return post(`/api/v1/enterprise/chat/${sessionId}`, payload, {
    headers: { 'X-Enterprise-Server-ID': serverId },
  })
}

export function proxyRetrieval(serverId: string, payload: Record<string, unknown>) {
  return post('/api/v1/enterprise/retrieval', payload, {
    headers: { 'X-Enterprise-Server-ID': serverId },
  })
}

export function proxySkillExecute(serverId: string, payload: Record<string, unknown>) {
  return post('/api/v1/enterprise/skill/execute', payload, {
    headers: { 'X-Enterprise-Server-ID': serverId },
  })
}
