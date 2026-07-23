<template>
  <div class="enterprise-server-manager">
    <div class="section-header">
      <h3>{{ $t('enterprise.title') }}</h3>
      <t-button v-if="!defaultServerRequired" size="small" theme="primary" @click="showAddDialog = true">
        <template #icon><t-icon name="add" /></template>
        {{ $t('enterprise.addServer') }}
      </t-button>
    </div>
    <p class="section-desc">{{ $t('enterprise.description') }}</p>

    <div v-if="store.loading" class="loading-state">
      <t-loading size="small" />
    </div>

    <div v-else-if="store.servers.length === 0" class="empty-state">
      <t-icon name="cloud" size="32" class="empty-icon" />
      <p>{{ $t('enterprise.noServers') }}</p>
    </div>

    <div v-else class="server-list">
      <div v-for="server in store.servers" :key="server.id" class="server-card">
        <div class="server-info">
          <div class="server-name-row">
            <span class="server-name">{{ server.name }}</span>
            <t-tag :theme="statusTheme(server.status)" size="small" variant="light">
              {{ statusLabel(server.status) }}
            </t-tag>
          </div>
          <span class="server-url">{{ server.base_url }}</span>
          <span v-if="server.linked_email" class="server-linked">
            {{ $t('enterprise.linkedAs', { email: server.linked_email }) }}
          </span>
          <span v-if="server.last_error" class="server-error">{{ server.last_error }}</span>
        </div>
        <div class="server-actions">
          <t-button
            v-if="server.status !== 'connected'"
            size="small"
            variant="text"
            theme="primary"
            @click="handleConnect(server.id)"
          >
            {{ $t('enterprise.connect') }}
          </t-button>
          <t-button
            v-else
            size="small"
            variant="text"
            theme="warning"
            @click="handleDisconnect(server.id)"
          >
            {{ $t('enterprise.disconnect') }}
          </t-button>
          <t-button size="small" variant="text" @click="handleTest(server.id)">
            {{ $t('enterprise.test') }}
          </t-button>
          <t-popconfirm v-if="!defaultServerRequired" :content="$t('enterprise.deleteConfirm')" @confirm="handleDelete(server.id)">
            <t-button size="small" variant="text" theme="danger">
              {{ $t('enterprise.delete') }}
            </t-button>
          </t-popconfirm>
        </div>
      </div>
    </div>

    <!-- Add Server Dialog -->
    <t-dialog
      v-model:visible="showAddDialog"
      :header="$t('enterprise.addServerTitle')"
      :confirm-btn="$t('enterprise.addServerConfirm')"
      :cancel-btn="$t('enterprise.cancel')"
      @confirm="handleAdd"
    >
      <t-form :label-width="100">
        <t-form-item :label="$t('enterprise.fieldName')">
          <t-input v-model="form.name" :placeholder="$t('enterprise.fieldNamePlaceholder')" />
        </t-form-item>
        <t-form-item :label="$t('enterprise.fieldURL')">
          <t-input v-model="form.base_url" placeholder="http://192.168.1.100:8080" />
        </t-form-item>
        <t-form-item :label="$t('auth.email')">
          <t-input v-model="form.email" autocomplete="email" :placeholder="$t('auth.emailPlaceholder')" />
        </t-form-item>
        <t-form-item :label="$t('auth.password')">
          <t-input v-model="form.password" type="password" :placeholder="$t('auth.passwordPlaceholder')" />
        </t-form-item>
        <t-form-item :label="$t('enterprise.fieldAutoConnect')">
          <t-switch v-model="form.auto_connect" />
        </t-form-item>
      </t-form>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'
import { useEnterpriseStore } from '@/stores/enterprise'
import { activateDesktopProfile, createDesktopProfile, loginDesktopProfile } from '@/api/desktop-remote'
import { useAuthStore } from '@/stores/auth'
import { useRuntimeContextStore } from '@/stores/runtimeContext'
import { getDesktopBootstrap } from '@/utils/api-context'

const { t } = useI18n()
const store = useEnterpriseStore()
const authStore = useAuthStore()
const runtimeStore = useRuntimeContextStore()
const defaultServerRequired = getDesktopBootstrap()?.default_enterprise_server_required === true

const showAddDialog = ref(false)
const form = reactive({
  name: '',
  base_url: '',
  email: '',
  password: '',
  auto_connect: true,
})

onMounted(() => {
  store.fetchServers()
})

function statusTheme(status?: string) {
  switch (status) {
    case 'connected': return 'success'
    case 'connecting': return 'warning'
    case 'error': return 'danger'
    default: return 'default'
  }
}

function statusLabel(status?: string) {
  switch (status) {
    case 'connected': return t('enterprise.statusConnected')
    case 'connecting': return t('enterprise.statusConnecting')
    case 'error': return t('enterprise.statusError')
    default: return t('enterprise.statusDisconnected')
  }
}

async function handleAdd() {
  if (!form.name || !form.base_url) {
    MessagePlugin.warning(t('enterprise.formIncomplete'))
    return
  }
  try {
    const shouldConnect = form.auto_connect
    const email = form.email
    const password = form.password
    const created = await createDesktopProfile({
      name: form.name,
      base_url: form.base_url,
      allow_insecure_transport: form.base_url.startsWith('http://'),
    })
    MessagePlugin.success(t('enterprise.addSuccess'))
    showAddDialog.value = false
    form.name = ''
    form.base_url = ''
    form.email = ''
    form.password = ''
    form.auto_connect = true
    await store.fetchServers()

    if (shouldConnect && created.id && email && password) {
      try {
        await loginAndSwitch(created.id, email, password)
        MessagePlugin.success(t('enterprise.connectSuccess'))
      } catch {
        MessagePlugin.error(t('enterprise.connectFailed'))
      }
    }
  } catch (e: unknown) {
    const message =
      e && typeof e === 'object'
        ? ('message' in e && typeof e.message === 'string' ? e.message : '')
        : ''
    MessagePlugin.error(message || (e instanceof Error ? e.message : t('enterprise.addFailed')))
    console.error('enterprise add failed:', e)
  }
}

async function handleConnect(id: string) {
  try {
    await store.connect(id)
    MessagePlugin.success(t('enterprise.connectSuccess'))
  } catch {
    MessagePlugin.error(t('enterprise.connectFailed'))
  }
}

async function handleDisconnect(id: string) {
  await store.disconnect(id)
}

async function handleTest(id: string) {
  try {
    await activateDesktopProfile(id)
    MessagePlugin.success(t('enterprise.testReachable'))
  } catch {
    MessagePlugin.error(t('enterprise.testFailed'))
  }
}

async function handleDelete(id: string) {
  try {
    await store.remove(id)
    MessagePlugin.success(t('enterprise.deleteSuccess'))
  } catch {
    MessagePlugin.error(t('enterprise.deleteFailed'))
  }
}

async function loginAndSwitch(id: string, email: string, password: string) {
  await activateDesktopProfile(id)
  const snapshot = await loginDesktopProfile(id, { email, password })
  await runtimeStore.useEnterprise(id, {
    userId: snapshot.user_id ?? null,
    tenantId: snapshot.tenant_id ?? null,
  })
  authStore.applyDesktopIdentitySnapshot(snapshot)
  await store.fetchServers()
}
</script>

<style scoped lang="less">
.enterprise-server-manager {
  padding: 16px 0;
}
.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
  h3 { margin: 0; font-size: 16px; }
}
.section-desc {
  color: var(--td-text-color-secondary);
  font-size: 13px;
  margin-bottom: 16px;
}
.loading-state, .empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 32px 0;
  color: var(--td-text-color-placeholder);
}
.empty-icon { margin-bottom: 8px; }
.server-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.server-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border: 1px solid var(--td-border-level-1-color);
  border-radius: 8px;
  background: var(--td-bg-color-container);
}
.server-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.server-name-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.server-name { font-weight: 500; }
.server-url {
  font-size: 12px;
  color: var(--td-text-color-secondary);
  font-family: monospace;
}
.server-linked {
  font-size: 12px;
  color: var(--td-success-color);
}
.server-error {
  font-size: 12px;
  color: var(--td-error-color);
}
.server-actions {
  display: flex;
  gap: 4px;
}
</style>
