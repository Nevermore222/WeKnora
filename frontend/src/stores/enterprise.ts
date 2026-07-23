import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import {
  activateDesktopProfile,
  deleteDesktopProfile,
  getDesktopProfileSession,
  listDesktopProfiles,
  logoutDesktopProfile,
  type DesktopServerProfile,
} from '@/api/desktop-remote'
import { useAuthStore } from '@/stores/auth'
import { useRuntimeContextStore } from '@/stores/runtimeContext'

export interface EnterpriseServer extends DesktopServerProfile {
  status?: 'connected' | 'connecting' | 'disconnected' | 'error'
  last_error?: string
  linked_email?: string
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

export const useEnterpriseStore = defineStore('enterprise', () => {
  const servers = ref<EnterpriseServer[]>([])
  const resources = ref<AggregatedResource[]>([])
  const loading = ref(false)
  const error = ref('')

  const connectedServers = computed(() =>
    servers.value.filter((s) => s.status === 'connected')
  )

  const hasConnection = computed(() => connectedServers.value.length > 0)

  const enterpriseKBs = computed(() =>
    resources.value.filter((r) => r.type === 'knowledge_base' && r.origin === 'enterprise')
  )

  const enterpriseAgents = computed(() =>
    resources.value.filter((r) => r.type === 'agent' && r.origin === 'enterprise')
  )

  const enterpriseSkills = computed(() =>
    resources.value.filter((r) => r.type === 'skill' && r.origin === 'enterprise')
  )

  async function withSession(profile: DesktopServerProfile): Promise<EnterpriseServer> {
    try {
      const session = await getDesktopProfileSession(profile.id)
      return {
        ...profile,
        status: session.authenticated ? 'connected' : 'disconnected',
        linked_email: typeof session.user?.email === 'string' ? session.user.email : undefined,
      }
    } catch (e) {
      return { ...profile, status: 'error', last_error: e instanceof Error ? e.message : String(e) }
    }
  }

  async function fetchServers() {
    loading.value = true
    error.value = ''
    try {
      const profiles = await listDesktopProfiles()
      servers.value = await Promise.all(profiles.map(withSession))
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'Failed to load servers'
    } finally {
      loading.value = false
    }
  }

  async function fetchResources(_type?: string) {
    resources.value = []
  }

  async function connect(id: string) {
    try {
      await activateDesktopProfile(id)
      const snapshot = await getDesktopProfileSession(id)
      if (!snapshot.authenticated) {
        throw new Error('desktop_login_required')
      }
      await useRuntimeContextStore().useEnterprise(id, {
        userId: snapshot.user_id ?? null,
        tenantId: snapshot.tenant_id ?? null,
      })
      useAuthStore().applyDesktopIdentitySnapshot(snapshot)
      await fetchServers()
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'Connection failed'
      throw e
    }
  }

  async function disconnect(id: string) {
    try {
      await logoutDesktopProfile(id)
      useAuthStore().clearDesktopIdentity()
      await useRuntimeContextStore().usePersonal()
      await fetchServers()
      resources.value = []
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'Disconnect failed'
    }
  }

  async function remove(id: string) {
    await deleteDesktopProfile(id)
    resources.value = []
    await fetchServers()
  }

  async function refresh() {
    await fetchServers()
  }

  return {
    servers,
    resources,
    loading,
    error,
    connectedServers,
    hasConnection,
    enterpriseKBs,
    enterpriseAgents,
    enterpriseSkills,
    fetchServers,
    fetchResources,
    connect,
    disconnect,
    remove,
    refresh,
  }
})
