<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue';
import { MessagePlugin } from 'tdesign-vue-next';
import { listWorkspaceFiles, type WorkspaceFileEntry } from '@/api/workspace';
import { useFilePreviewStore } from '@/stores/filePreview';
import { normalizeWorkspaceFileRef } from '@/utils/filePreview';

type BrowserNode = WorkspaceFileEntry & {
  children?: BrowserNode[];
  expanded?: boolean;
  loading?: boolean;
};

type BrowserRow = {
  node: BrowserNode;
  depth: number;
};

const props = defineProps<{
  workspaceId?: string;
  title?: string;
}>();

const store = useFilePreviewStore();
const query = ref('');
const rootNodes = ref<BrowserNode[]>([]);
const loadedDirs = reactive(new Set<string>(['']));
const loadingRoot = ref(false);

const visibleRows = computed<BrowserRow[]>(() => {
  const needle = query.value.trim().toLowerCase();
  if (needle) {
    return flattenNodes(rootNodes.value)
      .filter((node) => node.name.toLowerCase().includes(needle))
      .map((node) => ({ node, depth: 0 }));
  }
  return flattenTreeRows(rootNodes.value);
});

function flattenNodes(nodes: BrowserNode[]): BrowserNode[] {
  return nodes.flatMap((node) => [node, ...flattenNodes(node.children || [])]);
}

function flattenTreeRows(nodes: BrowserNode[], depth = 0): BrowserRow[] {
  return nodes.flatMap((node) => [
    { node, depth },
    ...(node.is_dir && node.expanded ? flattenTreeRows(node.children || [], depth + 1) : []),
  ]);
}

function sortEntries(entries: WorkspaceFileEntry[]) {
  return [...entries].sort((a, b) => {
    if (a.is_dir !== b.is_dir) return a.is_dir ? -1 : 1;
    return a.name.localeCompare(b.name);
  });
}

async function loadDir(path = '', target?: BrowserNode) {
  if (!props.workspaceId) {
    rootNodes.value = [];
    return;
  }

  if (target) {
    target.loading = true;
  } else {
    loadingRoot.value = true;
  }

  try {
    const response = await listWorkspaceFiles(props.workspaceId, path);
    const nodes = sortEntries(response?.files || []).map((file) => ({ ...file }));
    if (target) {
      target.children = nodes;
      target.expanded = true;
      loadedDirs.add(target.relative_path);
    } else {
      rootNodes.value = nodes;
      loadedDirs.add('');
    }
  } catch (error: any) {
    MessagePlugin.error(error?.message || '加载工作区文件失败');
  } finally {
    if (target) {
      target.loading = false;
    } else {
      loadingRoot.value = false;
    }
  }
}

async function toggleNode(node: BrowserNode) {
  if (!node.is_dir) {
    if (props.workspaceId) store.open(normalizeWorkspaceFileRef(props.workspaceId, node));
    return;
  }
  if (node.expanded) {
    node.expanded = false;
    return;
  }
  if (loadedDirs.has(node.relative_path) && node.children) {
    node.expanded = true;
    return;
  }
  await loadDir(node.relative_path, node);
}

function formatSize(size: number) {
  if (!Number.isFinite(size) || size <= 0) return '';
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / 1024 / 1024).toFixed(1)} MB`;
}

onMounted(() => loadDir(''));
watch(() => props.workspaceId, () => {
  loadedDirs.clear();
  loadedDirs.add('');
  query.value = '';
  loadDir('');
});
</script>

<template>
  <aside v-if="workspaceId" class="workspace-file-browser">
    <div class="browser-header">
      <div class="browser-title">
        <t-icon name="folder" size="16px" />
        <span>{{ title || '工作区文件' }}</span>
      </div>
      <div class="browser-actions">
        <t-button size="small" variant="text" :loading="loadingRoot" @click="loadDir('')">
          刷新
        </t-button>
        <t-button size="small" variant="text" shape="square" @click="store.closeBrowser">
          <t-icon name="close" />
        </t-button>
      </div>
    </div>

    <t-input v-model="query" class="browser-search" clearable placeholder="筛选文件...">
      <template #prefix-icon><t-icon name="search" /></template>
    </t-input>

    <t-loading :loading="loadingRoot" size="small">
      <div v-if="visibleRows.length" class="browser-tree">
        <template v-for="row in visibleRows" :key="row.node.relative_path">
          <button
            type="button"
            class="tree-row"
            :class="{ 'is-dir': row.node.is_dir, 'is-current': store.current?.relativePath === row.node.relative_path }"
            :style="{ paddingLeft: `${8 + row.depth * 18}px` }"
            @click="toggleNode(row.node)"
          >
            <t-icon
              :name="row.node.loading ? 'loading' : row.node.is_dir ? (row.node.expanded ? 'chevron-down' : 'chevron-right') : 'file'"
              size="15px"
            />
            <span class="tree-name">{{ row.node.name }}</span>
            <span class="tree-size">{{ row.node.is_dir ? '' : formatSize(row.node.size) }}</span>
          </button>
        </template>
      </div>
      <t-empty v-else class="browser-empty" description="工作区暂无文件" />
    </t-loading>
  </aside>
</template>

<style scoped>
.workspace-file-browser {
  position: fixed;
  top: 96px;
  right: 16px;
  bottom: 24px;
  z-index: 500;
  display: flex;
  width: min(360px, calc(100vw - 32px));
  padding: 14px;
  border: 1px solid color-mix(in srgb, var(--td-border-level-1-color) 70%, transparent);
  border-radius: 18px;
  background: color-mix(in srgb, var(--td-bg-color-container) 94%, transparent);
  box-shadow: 0 24px 80px rgba(0, 0, 0, 0.32);
  box-sizing: border-box;
  flex-direction: column;
  backdrop-filter: blur(18px);
}

.browser-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}

.browser-title {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 8px;
  color: var(--td-text-color-primary);
  font-size: 14px;
  font-weight: 700;
}

.browser-title span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.browser-actions {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  gap: 2px;
}

.browser-search {
  margin-bottom: 12px;
}

.browser-tree {
  display: flex;
  overflow: auto;
  max-height: calc(100vh - 210px);
  flex-direction: column;
  gap: 4px;
  padding-right: 2px;
}

.tree-row {
  display: flex;
  width: 100%;
  min-width: 0;
  align-items: center;
  gap: 8px;
  padding: 7px 8px;
  border: 1px solid transparent;
  border-radius: 9px;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  text-align: left;
}

.tree-row:hover,
.tree-row.is-current {
  border-color: color-mix(in srgb, var(--td-brand-color) 35%, transparent);
  background: color-mix(in srgb, var(--td-brand-color) 12%, transparent);
  color: var(--td-brand-color);
}

.tree-name {
  min-width: 0;
  flex: 1;
  overflow: hidden;
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tree-size {
  flex-shrink: 0;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
}

.browser-empty {
  margin-top: 32px;
}
</style>
