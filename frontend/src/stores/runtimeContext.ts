import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import {
  acceptResponse,
  getRuntimeContext,
  switchRuntimeContext,
  setRuntimeContext,
  type RuntimeContext,
} from '@/utils/api-context'
import { registerContextReset } from '@/utils/context-reset'

let defaultResetsRegistered = false

export const useRuntimeContextStore = defineStore('runtimeContext', () => {
  registerDefaultContextResets()

  const current = getRuntimeContext()
  const generation = ref(current.generation)
  const kind = ref<RuntimeContext['kind']>(current.kind)
  const profileId = ref(current.kind === 'enterprise' ? current.profileId : '')
  const userId = ref<string | null>(current.kind === 'enterprise' ? current.userId : null)
  const tenantId = ref<number | null>(current.kind === 'enterprise' ? current.tenantId : null)

  const context = computed<RuntimeContext>(() => {
    if (kind.value === 'enterprise') {
      return {
        kind: 'enterprise',
        profileId: profileId.value,
        userId: userId.value,
        tenantId: tenantId.value,
        generation: generation.value,
      }
    }
    return { kind: 'personal', generation: generation.value }
  })

  async function usePersonal() {
    const next = await switchRuntimeContext({ kind: 'personal', generation: generation.value })
    kind.value = next.kind
    generation.value = next.generation
    profileId.value = ''
    userId.value = null
    tenantId.value = null
  }

  async function useEnterprise(id: string, identity?: { userId?: string | null; tenantId?: number | null }) {
    const next = await switchRuntimeContext({
      kind: 'enterprise',
      profileId: id,
      userId: identity?.userId ?? null,
      tenantId: identity?.tenantId ?? null,
      generation: generation.value,
    })
    kind.value = next.kind
    generation.value = next.generation
    profileId.value = next.kind === 'enterprise' ? next.profileId : ''
    userId.value = next.kind === 'enterprise' ? next.userId : null
    tenantId.value = next.kind === 'enterprise' ? next.tenantId : null
  }

  function setEnterpriseTenant(nextTenantId: number | null) {
    if (kind.value !== 'enterprise') return
    const next: RuntimeContext = {
      kind: 'enterprise',
      profileId: profileId.value,
      userId: userId.value,
      tenantId: nextTenantId,
      generation: generation.value + 1,
    }
    setRuntimeContext(next)
    tenantId.value = nextTenantId
    generation.value = next.generation
  }

  function accepts(meta: { generation?: number } | null | undefined) {
    return acceptResponse(meta, generation.value)
  }

  return { context, kind, profileId, userId, tenantId, generation, usePersonal, useEnterprise, setEnterpriseTenant, accepts }
})

function registerDefaultContextResets() {
  if (defaultResetsRegistered) return
  defaultResetsRegistered = true
  registerContextReset(async () => {
    const [
      { useAuthStore },
      { useChatResourcesStore },
      { useEditorResourcesStore },
      { useOrganizationStore },
      { useSettingsStore },
    ] = await Promise.all([
      import('@/stores/auth'),
      import('@/stores/chatResources'),
      import('@/stores/editorResources'),
      import('@/stores/organization'),
      import('@/stores/settings'),
    ])
    useAuthStore().resetForRuntimeContext()
    useChatResourcesStore().invalidate()
    useEditorResourcesStore().invalidate()
    useOrganizationStore().clearState()
    useSettingsStore().resetForRuntimeContext()
  })
}
