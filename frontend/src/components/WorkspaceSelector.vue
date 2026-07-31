<template>
  <div class="workspace-selector">
    <div v-if="isPersonal" class="workspace-selector-row">
      <t-select
        v-model="selectedPath"
        class="workspace-select"
        clearable
        :loading="loading"
        :placeholder="$t('createChat.workspace.personalPlaceholder')"
        :popup-props="{ overlayClassName: 'workspace-select-popup' }"
      >
        <t-option
          v-for="item in recentItems"
          :key="item.root_path"
          :value="item.root_path"
          :label="item.workspace_name"
          :disabled="!!item.stale"
        >
          <div class="workspace-option">
            <t-icon name="folder" class="workspace-option-icon" />
            <div class="workspace-option-body">
              <span class="workspace-option-name">
                {{ item.workspace_name }}
                <span v-if="item.stale" class="workspace-stale">{{ $t('createChat.workspace.stale') }}</span>
              </span>
              <span class="workspace-option-path">{{ item.root_path }}</span>
            </div>
          </div>
        </t-option>
      </t-select>

      <t-tooltip :content="$t('createChat.workspace.openFolder')">
        <t-button shape="square" theme="primary" variant="outline" :loading="picking" @click="openFolder">
          <t-icon name="folder-open" />
        </t-button>
      </t-tooltip>

      <t-tooltip :content="$t('createChat.workspace.refresh')">
        <t-button shape="square" variant="outline" :loading="loading" @click="reloadRecent">
          <t-icon name="refresh" />
        </t-button>
      </t-tooltip>
    </div>

    <div v-else class="workspace-selector-row">
      <t-select
        v-model="selectedWorkspaceId"
        class="workspace-select"
        clearable
        :loading="loading"
        :placeholder="$t('createChat.workspace.placeholder')"
        :popup-props="{ overlayClassName: 'workspace-select-popup' }"
      >
        <t-option
          v-for="workspace in availableWorkspaces"
          :key="workspace.id"
          :value="workspace.id"
          :label="workspace.name"
        >
          <div class="workspace-option">
            <t-icon name="folder" class="workspace-option-icon" />
            <div class="workspace-option-body">
              <span class="workspace-option-name">{{ workspace.name }}</span>
              <span class="workspace-option-path">{{ workspace.relative_path }}</span>
            </div>
          </div>
        </t-option>
      </t-select>

      <t-tooltip :content="$t('createChat.workspace.refresh')">
        <t-button shape="square" variant="outline" :loading="loading" @click="loadWorkspaces">
          <t-icon name="refresh" />
        </t-button>
      </t-tooltip>

      <t-tooltip :content="$t('createChat.workspace.create')">
        <t-button shape="square" theme="primary" variant="outline" @click="showCreateDialog">
          <t-icon name="folder-add" />
        </t-button>
      </t-tooltip>
    </div>

    <p v-if="!isPersonal && unavailableCount > 0" class="workspace-note">
      {{ $t('createChat.workspace.unavailable', { count: unavailableCount }) }}
    </p>
    <p v-else-if="isPersonal" class="workspace-note">
      {{ $t('createChat.workspace.personalHint') }}
    </p>

    <t-dialog
      v-if="!isPersonal"
      v-model:visible="createVisible"
      width="420px"
      :header="$t('createChat.workspace.createTitle')"
      :confirm-btn="{ content: $t('createChat.workspace.createConfirm'), loading: creating, theme: 'primary' }"
      :cancel-btn="{ content: $t('common.cancel') }"
      :close-on-overlay-click="!creating"
      :close-on-esc-keydown="!creating"
      :on-confirm="handleCreate"
    >
      <t-input
        v-model="createName"
        :placeholder="$t('createChat.workspace.namePlaceholder')"
        :maxlength="128"
        autofocus
        @enter="handleCreate"
      />
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { MessagePlugin } from 'tdesign-vue-next'
import { createWorkspace, listWorkspaces, type WorkspaceEntry } from '@/api/workspace'
import { restoreWorkspaceSelection } from '@/utils/workspaceSelection'
import { useAuthStore } from '@/stores/auth'
import {
  loadRecentLocalWorkspaces,
  rememberRecentLocalWorkspace,
  type RecentLocalWorkspace,
} from '@/utils/recentLocalWorkspaces'

const LAST_WORKSPACE_KEY = 'xelora:last-workspace-id'
const LAST_LOCAL_PATH_KEY = 'xelora:last-local-workspace-path'

const props = defineProps<{
  modelValue: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
}>()

const { t } = useI18n()
const authStore = useAuthStore()
const isPersonal = computed(() => authStore.isPersonalMode)

const loading = ref(false)
const picking = ref(false)
const creating = ref(false)
const createVisible = ref(false)
const createName = ref('')
const workspaces = ref<WorkspaceEntry[]>([])
const recentItems = ref<RecentLocalWorkspace[]>([])

const availableWorkspaces = computed(() => workspaces.value.filter((entry) => entry.status === 'available'))
const unavailableCount = computed(() => workspaces.value.length - availableWorkspaces.value.length)

const selectedWorkspaceId = computed({
  get: () => props.modelValue,
  set: (value: string) => setSelection(value || ''),
})

const selectedPath = computed({
  get: () => props.modelValue,
  set: (value: string) => setLocalSelection(value || ''),
})

function normalizeWorkspaceList(response: any): WorkspaceEntry[] {
  if (Array.isArray(response)) return response
  if (Array.isArray(response?.data)) return response.data
  return []
}

function normalizeWorkspace(response: any): WorkspaceEntry | null {
  const entry = response?.data ?? response
  return entry && typeof entry.id === 'string' ? entry : null
}

function setSelection(workspaceId: string) {
  emit('update:modelValue', workspaceId)
  if (workspaceId) {
    localStorage.setItem(LAST_WORKSPACE_KEY, workspaceId)
  } else {
    localStorage.removeItem(LAST_WORKSPACE_KEY)
  }
}

function setLocalSelection(rootPath: string) {
  emit('update:modelValue', rootPath)
  if (rootPath) {
    localStorage.setItem(LAST_LOCAL_PATH_KEY, rootPath)
    const name = rootPath.replace(/[\\/]+$/, '').split(/[\\/]/).filter(Boolean).pop() || rootPath
    rememberRecentLocalWorkspace({
      workspace_id: '',
      workspace_name: name,
      root_path: rootPath,
    })
    reloadRecent()
  } else {
    localStorage.removeItem(LAST_LOCAL_PATH_KEY)
  }
}

function reloadRecent() {
  recentItems.value = loadRecentLocalWorkspaces()
}

async function openFolder() {
  picking.value = true
  try {
    const goApp = (window as any)?.go?.main?.App
    if (!goApp?.SelectDirectory) {
      MessagePlugin.warning(t('createChat.workspace.pickerUnavailable'))
      return
    }
    const path = await goApp.SelectDirectory()
    if (path) {
      setLocalSelection(path)
    }
  } catch (error: any) {
    console.error('[WorkspaceSelector] SelectDirectory failed:', error)
    MessagePlugin.error(error?.message || t('createChat.workspace.pickerFailed'))
  } finally {
    picking.value = false
  }
}

async function loadWorkspaces() {
  loading.value = true
  try {
    const entries = normalizeWorkspaceList(await listWorkspaces())
    workspaces.value = entries
    const restored = restoreWorkspaceSelection(
      props.modelValue || localStorage.getItem(LAST_WORKSPACE_KEY),
      entries,
    )
    if (restored !== props.modelValue) {
      setSelection(restored)
    }
  } catch (error: any) {
    console.error('[WorkspaceSelector] Failed to load workspaces:', error)
    MessagePlugin.error(error?.message || t('createChat.workspace.loadFailed'))
  } finally {
    loading.value = false
  }
}

function showCreateDialog() {
  createName.value = ''
  createVisible.value = true
}

async function handleCreate() {
  if (creating.value) return
  const name = createName.value.trim()
  if (!name) {
    MessagePlugin.warning(t('createChat.workspace.nameRequired'))
    return
  }

  creating.value = true
  try {
    const created = normalizeWorkspace(await createWorkspace(name))
    if (!created) {
      MessagePlugin.error(t('createChat.workspace.createFailed'))
      return
    }
    workspaces.value = [created, ...workspaces.value.filter((entry) => entry.id !== created.id)]
    setSelection(created.id)
    MessagePlugin.success(t('createChat.workspace.createSuccess'))
    createVisible.value = false
  } catch (error: any) {
    console.error('[WorkspaceSelector] Failed to create workspace:', error)
    MessagePlugin.error(error?.message || t('createChat.workspace.createFailed'))
  } finally {
    creating.value = false
  }
}

onMounted(() => {
  if (isPersonal.value) {
    reloadRecent()
    const last = props.modelValue || localStorage.getItem(LAST_LOCAL_PATH_KEY) || ''
    if (last && last !== props.modelValue) {
      emit('update:modelValue', last)
    }
  } else {
    loadWorkspaces()
  }
})
</script>

<style lang="less" scoped>
.workspace-selector {
  width: 100%;
  max-width: 800px;
  padding: 0 16px;
}

.workspace-selector-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.workspace-select {
  flex: 1;
  min-width: 0;
}

.workspace-note {
  margin: 8px 0 0;
  font-size: 12px;
  line-height: 1.5;
  color: var(--td-text-color-placeholder);
}

.workspace-option {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.workspace-option-icon {
  flex-shrink: 0;
  color: var(--td-brand-color);
}

.workspace-option-body {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.workspace-option-name,
.workspace-option-path {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.workspace-option-name {
  color: var(--td-text-color-primary);
}

.workspace-option-path {
  font-size: 12px;
  color: var(--td-text-color-placeholder);
}

.workspace-stale {
  margin-left: 6px;
  font-size: 11px;
  color: var(--td-warning-color);
}
</style>
