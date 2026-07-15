<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { MessagePlugin } from 'tdesign-vue-next';
import { listWorkspaceFiles, type WorkspaceFileEntry } from '@/api/workspace';
import { useFilePreviewStore } from '@/stores/filePreview';
import { normalizeWorkspaceFileRef } from '@/utils/filePreview';

const props = defineProps<{
  workspaceId?: string;
  title?: string;
}>();

const filePreviewStore = useFilePreviewStore();
const loading = ref(false);
const files = ref<WorkspaceFileEntry[]>([]);
const currentDir = ref('');

const visibleFiles = computed(() => files.value.slice(0, 8));

async function loadFiles(path = currentDir.value) {
  if (!props.workspaceId) {
    files.value = [];
    return;
  }
  loading.value = true;
  try {
    const response = await listWorkspaceFiles(props.workspaceId, path);
    files.value = Array.isArray(response?.files) ? response.files : [];
    currentDir.value = path;
  } catch (error: any) {
    files.value = [];
    MessagePlugin.error(error?.message || '加载工作区文件失败');
  } finally {
    loading.value = false;
  }
}

function openFile(file: WorkspaceFileEntry) {
  if (!props.workspaceId) return;
  if (file.is_dir) {
    loadFiles(file.relative_path);
    return;
  }
  filePreviewStore.open(normalizeWorkspaceFileRef(props.workspaceId, file));
}

function formatSize(size: number) {
  if (!Number.isFinite(size) || size <= 0) return '0 B';
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / 1024 / 1024).toFixed(1)} MB`;
}

onMounted(() => loadFiles(''));
watch(() => props.workspaceId, () => loadFiles(''));
</script>

<template>
  <section v-if="workspaceId" class="workspace-file-panel">
    <div class="workspace-file-panel__header">
      <div>
        <div class="workspace-file-panel__title">{{ title || '工作区文件' }}</div>
        <div v-if="currentDir" class="workspace-file-panel__path">{{ currentDir }}</div>
      </div>
      <div class="workspace-file-panel__actions">
        <t-button v-if="currentDir" size="small" variant="text" @click="loadFiles('')">根目录</t-button>
        <t-button size="small" variant="text" :loading="loading" @click="loadFiles(currentDir)">刷新</t-button>
      </div>
    </div>
    <t-loading :loading="loading" size="small">
      <div v-if="visibleFiles.length" class="workspace-file-panel__list">
        <button
          v-for="file in visibleFiles"
          :key="file.relative_path"
          type="button"
          class="workspace-file-panel__item"
          @click="openFile(file)"
        >
          <t-icon :name="file.is_dir ? 'folder' : 'file'" size="15px" />
          <span class="workspace-file-panel__name">{{ file.name }}</span>
          <span class="workspace-file-panel__meta">{{ file.is_dir ? '文件夹' : formatSize(file.size) }}</span>
        </button>
      </div>
      <t-empty v-else class="workspace-file-panel__empty" description="工作区暂无文件" />
    </t-loading>
  </section>
</template>

<style scoped>
.workspace-file-panel {
  width: 100%;
  margin-bottom: 8px;
  padding: 10px 12px;
  border: 1px solid var(--td-border-level-1-color);
  border-radius: 10px;
  background: color-mix(in srgb, var(--td-bg-color-container) 92%, transparent);
  box-sizing: border-box;
}

.workspace-file-panel__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 8px;
}

.workspace-file-panel__title {
  color: var(--td-text-color-primary);
  font-size: 12px;
  font-weight: 600;
}

.workspace-file-panel__path {
  max-width: 420px;
  overflow: hidden;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.workspace-file-panel__actions {
  display: flex;
  gap: 4px;
  flex-shrink: 0;
}

.workspace-file-panel__list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 6px;
}

.workspace-file-panel__item {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: 7px;
  padding: 6px 8px;
  border: 1px solid transparent;
  border-radius: 8px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-secondary);
  cursor: pointer;
  text-align: left;
}

.workspace-file-panel__item:hover {
  border-color: var(--td-brand-color-light);
  color: var(--td-brand-color);
}

.workspace-file-panel__name {
  min-width: 0;
  flex: 1;
  overflow: hidden;
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.workspace-file-panel__meta {
  flex-shrink: 0;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
}

.workspace-file-panel__empty {
  padding: 8px 0;
}

@media (max-width: 760px) {
  .workspace-file-panel__list {
    grid-template-columns: 1fr;
  }
}
</style>
