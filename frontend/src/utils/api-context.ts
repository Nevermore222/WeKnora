export type RuntimeContext =
  | { kind: 'personal'; generation: number }
  | {
      kind: 'enterprise'
      profileId: string
      userId: string | null
      tenantId: number | null
      generation: number
    }

export interface DesktopBootstrap {
  api_base_url: string
  session: string
  default_enterprise_server_url?: string
  default_enterprise_server_name?: string
  default_enterprise_server_required?: boolean
  default_enterprise_allow_insecure?: boolean
}

let bootstrap: DesktopBootstrap | null = null
let runtimeContext: RuntimeContext = { kind: 'personal', generation: 0 }

export function setDesktopBootstrap(value: DesktopBootstrap | null | undefined) {
  bootstrap = value
    ? {
        api_base_url: value.api_base_url || '',
        session: value.session || '',
        default_enterprise_server_url: value.default_enterprise_server_url || '',
        default_enterprise_server_name: value.default_enterprise_server_name || '',
        default_enterprise_server_required: value.default_enterprise_server_required === true,
        default_enterprise_allow_insecure: value.default_enterprise_allow_insecure === true,
      }
    : null
}

export function getDesktopBootstrap(): DesktopBootstrap | null {
  return bootstrap ? { ...bootstrap } : null
}

export function isDefaultEnterpriseRequired(): boolean {
  return bootstrap?.default_enterprise_server_required === true
}

export function setRuntimeContext(context: RuntimeContext) {
  runtimeContext = { ...context }
}

export async function switchRuntimeContext(destination: RuntimeContext): Promise<RuntimeContext> {
  const { resetActiveContext } = await import('./context-reset.ts')
  await resetActiveContext()

  const nextGeneration = runtimeContext.generation + 1
  const next =
    destination.kind === 'enterprise'
      ? {
          kind: 'enterprise' as const,
          profileId: destination.profileId,
          userId: destination.userId ?? null,
          tenantId: destination.tenantId ?? null,
          generation: nextGeneration,
        }
      : { kind: 'personal' as const, generation: nextGeneration }
  runtimeContext = next
  return { ...runtimeContext }
}

export function getRuntimeContext(): RuntimeContext {
  return { ...runtimeContext }
}

export function resetRuntimeContext() {
  runtimeContext = { kind: 'personal', generation: runtimeContext.generation + 1 }
}

export function acceptResponse(meta: { generation?: number } | null | undefined, currentGeneration: number): boolean {
  return meta?.generation === currentGeneration
}

export function resolveApiUrl(context: RuntimeContext, url: string): string {
  if (context.kind !== 'enterprise') {
    return url
  }

  const normalized = normalizeAPIPath(url)
  return `/desktop/remote/profiles/${encodeURIComponent(context.profileId)}${normalized}`
}

export function resolveCurrentApiUrl(url: string): string {
  return resolveApiUrl(runtimeContext, url)
}

export function desktopSessionHeader(): Record<string, string> {
  if (!bootstrap?.session) {
    return {}
  }
  return { 'X-Xelora-Desktop-Session': bootstrap.session }
}

export function desktopAPIBaseURL(): string {
  return bootstrap?.api_base_url || ''
}

function normalizeAPIPath(url: string): string {
  if (url.startsWith('/api/v1/')) {
    return url
  }
  if (url === '/api/v1') {
    return url
  }
  if (url.startsWith('api/v1/')) {
    return `/${url}`
  }
  if (url.startsWith('/')) {
    return `/api/v1${url}`
  }
  return `/api/v1/${url}`
}
