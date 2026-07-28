<template>
  <div class="user-permissions">
    <div class="section-header">
      <h2>{{ t('settings.userPermissions.title') }}</h2>
      <p class="section-description">{{ t('settings.userPermissions.description') }}</p>
    </div>

    <div v-if="loading" class="loading-inline">
      <t-loading size="small" />
    </div>

    <template v-else>
      <div class="settings-group">
        <div class="setting-row">
          <div class="setting-info">
            <label class="setting-label">{{ t('system.globalSettings.clientCapabilities.defaultsLabel') }}</label>
            <p class="desc">{{ t('system.globalSettings.clientCapabilities.defaultsDesc') }}</p>
          </div>
          <div class="setting-control caps-switches">
            <label class="cap-switch">
              <span>{{ t('capability.canCreateKnowledgeBase') }}</span>
              <t-switch v-model="defaults.can_create_knowledge_base" @change="persistDefaults" />
            </label>
            <label class="cap-switch">
              <span>{{ t('capability.canCreateAgent') }}</span>
              <t-switch v-model="defaults.can_create_agent" @change="persistDefaults" />
            </label>
            <label class="cap-switch">
              <span>{{ t('capability.canBindModel') }}</span>
              <t-switch v-model="defaults.can_bind_model" @change="persistDefaults" />
            </label>
          </div>
        </div>

        <div class="setting-row">
          <div class="setting-info">
            <label class="setting-label">{{ t('system.globalSettings.clientCapabilities.tightenLabel') }}</label>
            <p class="desc">{{ t('system.globalSettings.clientCapabilities.tightenDesc') }}</p>
          </div>
          <div class="setting-control">
            <t-switch v-model="tightenFlag" @change="persistDefaults" />
            <t-popconfirm
              :content="t('system.globalSettings.clientCapabilities.tightenConfirm')"
              :confirm-btn="t('system.globalSettings.clientCapabilities.tightenRun')"
              :cancel-btn="t('common.cancel')"
              @confirm="runTighten"
            >
              <t-button size="small" variant="outline" :loading="tightening" style="margin-left: 12px">
                {{ t('system.globalSettings.clientCapabilities.tightenRun') }}
              </t-button>
            </t-popconfirm>
          </div>
        </div>
      </div>

      <div class="users-toolbar">
        <t-input
          v-model="searchQ"
          clearable
          :placeholder="t('settings.userPermissions.searchPlaceholder')"
          style="max-width: 260px"
          @change="onSearchChange"
        />
        <t-select v-model="filter" :options="filterOptions" style="width: 180px" @change="onFilterChange" />
      </div>

      <t-table
        row-key="id"
        :data="users"
        :columns="columns"
        size="small"
        hover
        :loading="usersLoading"
      >
        <template #user="{ row }">
          <div class="user-cell">
            <span class="user-name">{{ row.username || row.email || row.id }}</span>
            <span v-if="row.username && row.email" class="user-email">{{ row.email }}</span>
          </div>
        </template>
        <template #badges="{ row }">
          <div class="badge-cell">
            <t-tag v-if="row.is_client_provisioned" size="small" variant="light">
              {{ t('settings.userPermissions.badgeClient') }}
            </t-tag>
            <t-tag v-if="row.is_system_admin" size="small" theme="warning" variant="light">
              {{ t('settings.userPermissions.badgeSystemAdmin') }}
            </t-tag>
            <span v-if="!row.is_client_provisioned && !row.is_system_admin" class="badge-empty">—</span>
          </div>
        </template>
        <template #tenant="{ row }">
          <span>{{ row.home_tenant_name || '—' }}</span>
        </template>
        <template #can_kb="{ row }">
          <t-switch
            :value="!!row.can_create_knowledge_base"
            size="small"
            @change="(v: boolean) => onUserCap(row, 'can_create_knowledge_base', v)"
          />
        </template>
        <template #can_agent="{ row }">
          <t-switch
            :value="!!row.can_create_agent"
            size="small"
            @change="(v: boolean) => onUserCap(row, 'can_create_agent', v)"
          />
        </template>
        <template #can_model="{ row }">
          <t-switch
            :value="!!row.can_bind_model"
            size="small"
            @change="(v: boolean) => onUserCap(row, 'can_bind_model', v)"
          />
        </template>
      </t-table>

      <div v-if="total > 0" class="pager">
        <t-pagination
          v-model="page"
          v-model:page-size="pageSize"
          :total="total"
          size="small"
          show-jumper
          show-page-number
          show-page-size
          :page-size-options="[20, 50, 100]"
          @change="onPageChange"
        />
      </div>

      <div v-else-if="!usersLoading" class="empty-hint">
        {{ t('settings.userPermissions.empty') }}
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'
import {
  getClientCapabilityDefaults,
  updateClientCapabilityDefaults,
  tightenExistingClientUsers,
  listAdminUsers,
  updateUserCapabilities,
  type AdminUserFilter,
  type AdminUserRow,
} from '@/api/system/capabilities'

const { t } = useI18n()

const loading = ref(true)
const usersLoading = ref(false)
const tightening = ref(false)
const defaults = reactive({
  can_create_knowledge_base: false,
  can_create_agent: false,
  can_bind_model: false,
})
const tightenFlag = ref(false)
const users = ref<AdminUserRow[]>([])
const total = ref(0)
const searchQ = ref('')
const filter = ref<AdminUserFilter>('all')
const page = ref(1)
const pageSize = ref(50)

let searchTimer: ReturnType<typeof setTimeout> | null = null

const filterOptions = computed(() => [
  { label: t('settings.userPermissions.filterAll'), value: 'all' },
  { label: t('settings.userPermissions.filterClient'), value: 'client' },
  { label: t('settings.userPermissions.filterNonClient'), value: 'non_client' },
  { label: t('settings.userPermissions.filterSystemAdmin'), value: 'system_admin' },
])

const columns = computed(() => [
  { colKey: 'user', title: t('settings.userPermissions.colUser'), minWidth: 160 },
  { colKey: 'badges', title: t('settings.userPermissions.colBadges'), width: 160 },
  { colKey: 'tenant', title: t('settings.userPermissions.colTenant'), minWidth: 140, ellipsis: true },
  { colKey: 'can_kb', title: t('capability.canCreateKnowledgeBase'), width: 110 },
  { colKey: 'can_agent', title: t('capability.canCreateAgent'), width: 110 },
  { colKey: 'can_model', title: t('capability.canBindModel'), width: 110 },
])

async function loadDefaults() {
  const resp = await getClientCapabilityDefaults()
  defaults.can_create_knowledge_base = !!resp.defaults?.can_create_knowledge_base
  defaults.can_create_agent = !!resp.defaults?.can_create_agent
  defaults.can_bind_model = !!resp.defaults?.can_bind_model
  tightenFlag.value = !!resp.tighten_existing_client_users
}

async function loadUsers() {
  usersLoading.value = true
  try {
    const resp = await listAdminUsers({
      q: searchQ.value.trim() || undefined,
      filter: filter.value,
      offset: (page.value - 1) * pageSize.value,
      limit: pageSize.value,
    })
    users.value = resp.users ?? []
    total.value = resp.total ?? 0
  } catch (e: any) {
    MessagePlugin.error(e?.message || t('system.globalSettings.clientCapabilities.loadFailed'))
  } finally {
    usersLoading.value = false
  }
}

function onSearchChange() {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    page.value = 1
    loadUsers()
  }, 300)
}

function onFilterChange() {
  page.value = 1
  loadUsers()
}

function onPageChange() {
  loadUsers()
}

async function persistDefaults() {
  try {
    await updateClientCapabilityDefaults({
      can_create_knowledge_base: defaults.can_create_knowledge_base,
      can_create_agent: defaults.can_create_agent,
      can_bind_model: defaults.can_bind_model,
      tighten_existing_client_users: tightenFlag.value,
    })
    MessagePlugin.success(t('system.globalSettings.clientCapabilities.saved'))
  } catch (e: any) {
    MessagePlugin.error(e?.message || t('system.globalSettings.clientCapabilities.saveFailed'))
  }
}

async function runTighten() {
  tightening.value = true
  try {
    const resp = await tightenExistingClientUsers()
    MessagePlugin.success(
      t('system.globalSettings.clientCapabilities.tightenDone', { n: resp.updated_count ?? 0 }),
    )
    await loadUsers()
  } catch (e: any) {
    MessagePlugin.error(e?.message || t('system.globalSettings.clientCapabilities.saveFailed'))
  } finally {
    tightening.value = false
  }
}

async function onUserCap(
  row: AdminUserRow,
  key: 'can_create_knowledge_base' | 'can_create_agent' | 'can_bind_model',
  value: boolean,
) {
  const next = {
    can_create_knowledge_base: !!row.can_create_knowledge_base,
    can_create_agent: !!row.can_create_agent,
    can_bind_model: !!row.can_bind_model,
    [key]: value,
  }
  try {
    const updated = await updateUserCapabilities(row.id, next)
    Object.assign(row, updated)
  } catch (e: any) {
    MessagePlugin.error(e?.message || t('system.globalSettings.clientCapabilities.saveFailed'))
    await loadUsers()
  }
}

onMounted(async () => {
  loading.value = true
  try {
    await Promise.all([loadDefaults(), loadUsers()])
  } catch (e: any) {
    MessagePlugin.error(e?.message || t('system.globalSettings.clientCapabilities.loadFailed'))
  } finally {
    loading.value = false
  }
})
</script>

<style scoped lang="less">
.user-permissions {
  .section-header {
    margin-bottom: 20px;

    h2 {
      margin: 0 0 6px;
      font-size: 18px;
      font-weight: 600;
    }

    .section-description {
      margin: 0;
      color: var(--td-text-color-secondary);
      font-size: 13px;
      line-height: 1.5;
    }
  }
}

.loading-inline {
  display: flex;
  padding: 16px 0;
}

.settings-group {
  margin-bottom: 20px;
}

.setting-row {
  display: flex;
  justify-content: space-between;
  gap: 24px;
  padding: 14px 0;
  border-bottom: 1px solid var(--td-component-border);
}

.setting-info {
  flex: 1;
  min-width: 0;

  .setting-label {
    font-weight: 500;
    font-size: 14px;
  }

  .desc {
    margin: 4px 0 0;
    font-size: 12px;
    color: var(--td-text-color-secondary);
  }
}

.setting-control {
  display: flex;
  align-items: center;
  flex-shrink: 0;
}

.caps-switches {
  display: flex;
  flex-direction: column;
  gap: 10px;
  align-items: flex-end;
}

.cap-switch {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 13px;
}

.users-toolbar {
  display: flex;
  gap: 12px;
  margin: 16px 0 12px;
  flex-wrap: wrap;
}

.user-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.user-name {
  font-weight: 500;
}

.user-email {
  font-size: 12px;
  color: var(--td-text-color-secondary);
}

.badge-cell {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.badge-empty {
  color: var(--td-text-color-placeholder);
}

.pager {
  margin-top: 12px;
  display: flex;
  justify-content: flex-end;
}

.empty-hint {
  padding: 24px 0;
  text-align: center;
  color: var(--td-text-color-secondary);
  font-size: 13px;
}
</style>
