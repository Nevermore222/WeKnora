import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getRuntimeContext } from '../utils/api-context.ts'
import { registerContextReset } from '../utils/context-reset.ts'

export type ClientFeatureRuntimeKind = 'personal' | 'enterprise'

export interface ClientFeaturesResponseShape {
  features?: {
    workspace_binding?: boolean
  }
}

export interface ClientFeaturesStoreOptions {
  kind?: ClientFeatureRuntimeKind | (() => ClientFeatureRuntimeKind)
  fetchFeatures?: () => Promise<ClientFeaturesResponseShape>
}

function defaultRuntimeKind(): ClientFeatureRuntimeKind {
  return getRuntimeContext().kind
}

async function fetchClientFeatures(): Promise<ClientFeaturesResponseShape> {
  const { getClientFeatures } = await import('../api/system/capabilities.ts')
  return getClientFeatures()
}

export function createClientFeaturesController(options: ClientFeaturesStoreOptions = {}) {
  const loaded = ref(false)
  const workspaceBindingEnabled = ref(false)
  const resolveKind = typeof options.kind === 'function'
    ? options.kind
    : () => options.kind ?? defaultRuntimeKind()
  const fetchFeatures = options.fetchFeatures ?? fetchClientFeatures

  async function refresh() {
    workspaceBindingEnabled.value = false
    if (resolveKind() !== 'enterprise') {
      loaded.value = true
      return
    }

    loaded.value = false
    try {
      const response = await fetchFeatures()
      workspaceBindingEnabled.value = response?.features?.workspace_binding === true
    } catch {
      workspaceBindingEnabled.value = false
    } finally {
      loaded.value = true
    }
  }

  function reset() {
    loaded.value = false
    workspaceBindingEnabled.value = false
  }

  return { loaded, workspaceBindingEnabled, refresh, reset }
}

export function createClientFeaturesStoreForTest(options: ClientFeaturesStoreOptions) {
  const controller = createClientFeaturesController(options)
  return {
    get loaded() { return controller.loaded.value },
    get workspaceBindingEnabled() { return controller.workspaceBindingEnabled.value },
    refresh: controller.refresh,
    reset: controller.reset,
  }
}

export const useClientFeaturesStore = defineStore('clientFeatures', () => {
  const controller = createClientFeaturesController()
  registerContextReset(controller.reset)
  return controller
})
