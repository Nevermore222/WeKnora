<template>
  <div class="workspace-selector">
    <div class="workspace-selector-row">
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

    <p v-if="unavailableCount > 0" class="workspace-note">
      {{ $t('createChat.workspace.unavailable', { count: unavailableCount }) }}
    </p>

    <t-dialog
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
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { MessagePlugin } from 'tdesign-vue-next';
import { createWorkspace, listWorkspaces, type WorkspaceEntry } from '@/api/workspace';
import { restoreWorkspaceSelection } from '@/utils/workspaceSelection';

const LAST_WORKSPACE_KEY = 'xelora:last-workspace-id';

const props = defineProps<{
  modelValue: string;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void;
}>();

const { t } = useI18n();

const loading = ref(false);
const creating = ref(false);
const createVisible = ref(false);
const createName = ref('');
const workspaces = ref<WorkspaceEntry[]>([]);

const availableWorkspaces = computed(() => workspaces.value.filter((entry) => entry.status === 'available'));
const unavailableCount = computed(() => workspaces.value.length - availableWorkspaces.value.length);

const selectedWorkspaceId = computed({
  get: () => props.modelValue,
  set: (value: string) => setSelection(value || ''),
});

function normalizeWorkspaceList(response: any): WorkspaceEntry[] {
  if (Array.isArray(response)) return response;
  if (Array.isArray(response?.data)) return response.data;
  return [];
}

function normalizeWorkspace(response: any): WorkspaceEntry | null {
  const entry = response?.data ?? response;
  return entry && typeof entry.id === 'string' ? entry : null;
}

function setSelection(workspaceId: string) {
  emit('update:modelValue', workspaceId);
  if (workspaceId) {
    localStorage.setItem(LAST_WORKSPACE_KEY, workspaceId);
  } else {
    localStorage.removeItem(LAST_WORKSPACE_KEY);
  }
}

async function loadWorkspaces() {
  loading.value = true;
  try {
    const entries = normalizeWorkspaceList(await listWorkspaces());
    workspaces.value = entries;
    const restored = restoreWorkspaceSelection(
      props.modelValue || localStorage.getItem(LAST_WORKSPACE_KEY),
      entries,
    );
    if (restored !== props.modelValue) {
      setSelection(restored);
    }
  } catch (error: any) {
    console.error('[WorkspaceSelector] Failed to load workspaces:', error);
    MessagePlugin.error(error?.message || t('createChat.workspace.loadFailed'));
  } finally {
    loading.value = false;
  }
}

function showCreateDialog() {
  createName.value = '';
  createVisible.value = true;
}

async function handleCreate() {
  if (creating.value) return;
  const name = createName.value.trim();
  if (!name) {
    MessagePlugin.warning(t('createChat.workspace.nameRequired'));
    return;
  }

  creating.value = true;
  try {
    const created = normalizeWorkspace(await createWorkspace(name));
    if (!created) {
      MessagePlugin.error(t('createChat.workspace.createFailed'));
      return;
    }
    workspaces.value = [
      created,
      ...workspaces.value.filter((entry) => entry.id !== created.id),
    ];
    setSelection(created.id);
    MessagePlugin.success(t('createChat.workspace.createSuccess'));
    createVisible.value = false;
  } catch (error: any) {
    console.error('[WorkspaceSelector] Failed to create workspace:', error);
    MessagePlugin.error(error?.message || t('createChat.workspace.createFailed'));
  } finally {
    creating.value = false;
  }
}

onMounted(() => {
  loadWorkspaces();
});
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
</style>
