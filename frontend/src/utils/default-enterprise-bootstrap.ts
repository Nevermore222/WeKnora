import type { RuntimeContext } from './api-context.ts'
import type { DesktopServerProfile } from '@/api/desktop-remote'

export interface DefaultEnterpriseBootstrap {
  default_enterprise_server_url?: string
  default_enterprise_server_name?: string
  default_enterprise_allow_insecure?: boolean
}

export interface DefaultEnterpriseBootstrapDeps {
  ensureProfile: (config: {
    name?: string
    base_url: string
    allow_insecure_transport?: boolean
  }) => Promise<DesktopServerProfile>
  setContext: (context: RuntimeContext) => void
  delay?: (ms: number) => Promise<void>
}

export interface DefaultEnterpriseBootstrapOptions {
  attempts?: number
  delayMs?: number
}

export async function initializeDefaultEnterpriseProfile(
  bootstrap: DefaultEnterpriseBootstrap | null | undefined,
  deps: DefaultEnterpriseBootstrapDeps,
  options: DefaultEnterpriseBootstrapOptions = {},
): Promise<RuntimeContext | null> {
  const baseURL = String(bootstrap?.default_enterprise_server_url || '').trim()
  if (!baseURL) return null

  const attempts = Math.max(1, options.attempts ?? 8)
  const delayMs = Math.max(0, options.delayMs ?? 250)
  const wait = deps.delay ?? defaultDelay
  let lastError: unknown = null

  for (let attempt = 1; attempt <= attempts; attempt += 1) {
    try {
      const profile = await deps.ensureProfile({
        name: String(bootstrap?.default_enterprise_server_name || 'Xelora Server'),
        base_url: baseURL,
        allow_insecure_transport: bootstrap?.default_enterprise_allow_insecure === true,
      })
      const context: RuntimeContext = {
        kind: 'enterprise',
        profileId: profile.id,
        userId: null,
        tenantId: null,
        generation: 0,
      }
      deps.setContext(context)
      return context
    } catch (error) {
      lastError = error
      if (attempt < attempts) {
        await wait(delayMs)
      }
    }
  }

  throw lastError instanceof Error ? lastError : new Error('default enterprise profile initialization failed')
}

function defaultDelay(ms: number) {
  return new Promise<void>((resolve) => window.setTimeout(resolve, ms))
}
