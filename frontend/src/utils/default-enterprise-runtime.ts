import {
  getDesktopBootstrap,
  getRuntimeContext,
  setRuntimeContext,
  switchRuntimeContext,
  type RuntimeContext,
} from './api-context.ts'
import {
  initializeDefaultEnterpriseProfile,
  type DefaultEnterpriseBootstrap,
} from './default-enterprise-bootstrap.ts'

export interface DefaultEnterpriseRuntimeControllerDeps {
  getBootstrap: () => DefaultEnterpriseBootstrap | null
  getContext: () => RuntimeContext
  initialize: (bootstrap: DefaultEnterpriseBootstrap) => Promise<RuntimeContext | null>
  switchContext: (context: RuntimeContext) => Promise<RuntimeContext>
}

export interface EnsureDefaultEnterpriseRuntimeOptions {
  switchContext?: boolean
}

export interface DefaultEnterpriseRuntimeController {
  ensure: (options?: EnsureDefaultEnterpriseRuntimeOptions) => Promise<RuntimeContext | null>
}

export function createDefaultEnterpriseRuntimeController(
  deps: DefaultEnterpriseRuntimeControllerDeps,
): DefaultEnterpriseRuntimeController {
  let activeAttempt: Promise<RuntimeContext | null> | null = null

  const ensure = async (
    options: EnsureDefaultEnterpriseRuntimeOptions = {},
  ): Promise<RuntimeContext | null> => {
    const current = deps.getContext()
    if (current.kind === 'enterprise') return current

    const bootstrap = deps.getBootstrap()
    if (!bootstrap?.default_enterprise_server_url) return null

    if (!activeAttempt) {
      activeAttempt = deps.initialize(bootstrap).finally(() => {
        activeAttempt = null
      })
    }

    const context = await activeAttempt
    if (options.switchContext === true && context?.kind === 'enterprise') {
      return deps.switchContext(context)
    }
    return context
  }

  return { ensure }
}

async function initializeWithDefaultDesktopProfile(
  bootstrap: DefaultEnterpriseBootstrap,
): Promise<RuntimeContext | null> {
  const { ensureDefaultDesktopProfile } = await import('../api/desktop-remote/index.ts')
  return initializeDefaultEnterpriseProfile(bootstrap, {
    ensureProfile: ensureDefaultDesktopProfile,
    setContext: setRuntimeContext,
  })
}

const defaultEnterpriseRuntimeController = createDefaultEnterpriseRuntimeController({
  getBootstrap: getDesktopBootstrap,
  getContext: getRuntimeContext,
  initialize: initializeWithDefaultDesktopProfile,
  switchContext: switchRuntimeContext,
})

export function ensureDefaultEnterpriseRuntime(
  options: EnsureDefaultEnterpriseRuntimeOptions = {},
): Promise<RuntimeContext | null> {
  return defaultEnterpriseRuntimeController.ensure(options)
}
