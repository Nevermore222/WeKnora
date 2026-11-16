import { del, get, post, put } from '@/utils/request'

export type McpEndpointToolGroup = 'retrieve' | 'chat' | 'wiki' | 'ingest'

export interface McpEndpoint {
  id: string
  tenant_id: number
  name: string
  description: string
  enabled: boolean
  token_hint: string
  knowledge_base_ids: string[]
  tools: string[]
  default_agent_id: string
  rate_limit_per_minute: number
  path: string
  last_used_at?: string | null
  created_at: string
  updated_at: string
  token?: string
}

export interface McpEndpointToolDefinition {
  name: string
  group: McpEndpointToolGroup
  destructive: boolean
}

export interface McpEndpointToolCatalog {
  groups: McpEndpointToolGroup[]
  tools: McpEndpointToolDefinition[]
  default_tools: string[]
}

export interface McpEndpointPayload {
  name?: string
  description?: string
  enabled?: boolean
  knowledge_base_ids?: string[]
  tools?: string[]
  default_agent_id?: string
  rate_limit_per_minute?: number
}

export async function listMcpEndpoints(): Promise<McpEndpoint[]> {
  const response: any = await get('/api/v1/mcp-endpoints')
  return response.data || []
}
export async function getMcpEndpointToolCatalog(): Promise<McpEndpointToolCatalog> {
  const response: any = await get('/api/v1/mcp-endpoints/tools')
  return response.data
}
export async function createMcpEndpoint(data: McpEndpointPayload): Promise<McpEndpoint> {
  const response: any = await post('/api/v1/mcp-endpoints', data)
  return response.data
}
export async function updateMcpEndpoint(id: string, data: McpEndpointPayload): Promise<McpEndpoint> {
  const response: any = await put(`/api/v1/mcp-endpoints/${id}`, data)
  return response.data
}
export async function deleteMcpEndpoint(id: string): Promise<void> { await del(`/api/v1/mcp-endpoints/${id}`) }
export async function rotateMcpEndpointToken(id: string): Promise<McpEndpoint> {
  const response: any = await post(`/api/v1/mcp-endpoints/${id}/rotate-token`, {})
  return response.data
}
