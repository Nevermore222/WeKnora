<template>
  <div class="client-caps-section">
    <div class="settings-group">
      <div class="setting-row setting-row--header">
        <div class="setting-info">
          <label class="setting-label">
            <span>{{ t('system.globalSettings.clientCapabilities.title') }}</span>
          </label>
          <p class="desc">{{ t('system.globalSettings.clientCapabilities.description') }}</p>
        </div>
      </div>

      <div v-if="loading" class="loading-inline">
        <t-loading size="small" />
      </div>

      <template v-else>
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

        <div class="setting-row setting-row--table">
          <div class="setting-info">
            <label class="setting-label">{{ t('system.globalSettings.clientCapabilities.usersLabel') }}</label>
            <p class="desc">{{ t('system.globalSettings.clientCapabilities.usersDesc', { n: total }) }}</p>
          </div>
          <div class="client-users-table">
            <t-table row-key="id" :data="users" :columns="columns" size="small" hover :loading="usersLoading">
              <template #user="{ row }">
                <div class="user-cell">
                  <span class="user-name">{{ row.username || row.email || row.id }}</span>
                  <span v-if="row.username && row.email" class="user-email">{{ row.email }}</span>
                </div>
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
          </div>
        </div>
      </template>
    </div>
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
  listClientUsers,
  updateUserCapabilities,
  type ClientUserRow,
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
const users = ref<ClientUserRow[]>([])
const total = ref(0)

const columns = computed(() => [
  { colKey: 'user', title: t('system.globalSettings.clientCapabilities.colUser'), minWidth: 160 },
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
    const resp = await listClientUsers({ offset: 0, limit: 100 })
    users.value = resp.users ?? []
    total.value = resp.total ?? 0
  } finally {
    usersLoading.value = false
  }
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
  row: ClientUserRow,
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
.client-caps-section {
  margin-bottom: 24px;
}
.loading-inline {
  display: flex;
  padding: 16px 0;
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
  color: var(--td-text-color-primary);
}
.setting-row--table {
  flex-direction: column;
  align-items: stretch;
}
.client-users-table {
  width: 100%;
  margin-top: 8px;
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
</style>
