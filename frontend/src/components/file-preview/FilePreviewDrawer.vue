<script setup lang="ts">
import { computed } from 'vue';
import { useFilePreviewStore } from '@/stores/filePreview';
import { MessagePlugin } from 'tdesign-vue-next';
import { openPreviewFile } from '@/utils/filePreview';
import FilePreviewActions from './FilePreviewActions.vue';
import FilePreviewBody from './FilePreviewBody.vue';

const store = useFilePreviewStore();
const file = computed(() => store.current);

const breadcrumbs = computed(() => {
  const path = file.value?.relativePath || file.value?.path || '';
  const parts = path.split('/').filter(Boolean);
  return parts.length ? parts : file.value ? [file.value.name] : [];
});

async function openRawFile() {
  if (!file.value?.workspaceId) return;
  try {
    await openPreviewFile(file.value);
  } catch (error: any) {
    MessagePlugin.error(error?.message || '打开文件失败');
  }
}

function formatSize(size?: number) {
  if (!Number.isFinite(size || 0) || !size) return '';
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / 1024 / 1024).toFixed(1)} MB`;
}
</script>

<template>
  <teleport to="body">
    <transition name="file-preview-fade">
      <section
        v-if="store.visible && file"
        class="file-preview-workbench"
        :class="{ 'has-browser': store.browserVisible }"
      >
        <header class="file-workbench-tabs">
          <div class="file-tabs">
            <div
              v-for="opened in store.openedFiles"
              :key="`${opened.workspaceId || opened.sessionId || opened.source}:${opened.relativePath || opened.path}`"
              role="button"
              tabindex="0"
              class="file-tab"
              :class="{ 'is-active': store.currentKey === `${opened.workspaceId || opened.sessionId || opened.source}:${opened.relativePath || opened.path}` }"
              @click="store.activate(opened)"
              @keydown.enter.prevent="store.activate(opened)"
              @keydown.space.prevent="store.activate(opened)"
            >
              <t-icon :name="opened.kind === 'image' ? 'image' : 'file'" size="15px" />
              <span>{{ opened.name }}</span>
              <button type="button" class="file-tab-close" title="关闭" @click.stop="store.closeFile(opened)">
                <t-icon name="close" size="13px" />
              </button>
            </div>
          </div>
          <button type="button" class="file-tab-add" title="从右侧文件树打开文件" @click="store.openBrowser">
            <t-icon name="add" size="16px" />
          </button>
        </header>

        <div class="file-workbench-toolbar">
          <div class="file-breadcrumbs">
            <template v-for="(part, index) in breadcrumbs" :key="`${part}-${index}`">
              <span class="breadcrumb-item" :class="{ 'is-current': index === breadcrumbs.length - 1 }">{{ part }}</span>
              <t-icon v-if="index < breadcrumbs.length - 1" name="chevron-right" size="13px" />
            </template>
          </div>
          <div class="file-toolbar-actions">
            <span class="file-meta">{{ file.kind || 'file' }}</span>
            <span v-if="file.size != null" class="file-meta">{{ formatSize(file.size) }}</span>
            <t-button size="small" variant="outline" :disabled="!file.workspaceId" @click="openRawFile">
              打开
            </t-button>
            <t-button size="small" variant="text" shape="square" @click="store.close">
              <t-icon name="close" />
            </t-button>
          </div>
        </div>

        <div class="file-workbench-main">
          <aside class="file-workbench-info">
            <div class="file-kind-badge">{{ file.kind || 'file' }}</div>
            <div class="file-info-name">{{ file.name }}</div>
            <div class="file-info-path">{{ file.relativePath || file.path }}</div>
            <FilePreviewActions :file="file" />
          </aside>
          <main class="file-preview-content">
            <div class="file-preview-surface">
              <FilePreviewBody :file="file" />
            </div>
          </main>
        </div>
      </section>
    </transition>
  </teleport>
</template>

<style scoped>
.file-preview-workbench {
  position: fixed;
  top: 96px;
  right: 56px;
  bottom: 24px;
  left: 360px;
  z-index: 480;
  display: flex;
  overflow: hidden;
  border: 1px solid color-mix(in srgb, var(--td-border-level-1-color) 72%, transparent);
  border-radius: 18px;
  background: color-mix(in srgb, var(--td-bg-color-page) 92%, var(--td-bg-color-container));
  box-shadow: 0 24px 80px rgba(0, 0, 0, 0.30);
  flex-direction: column;
  backdrop-filter: blur(18px);
}

.file-preview-workbench.has-browser {
  right: 396px;
}

.file-workbench-tabs {
  display: flex;
  min-height: 48px;
  align-items: center;
  gap: 10px;
  padding: 8px 12px 0;
  border-bottom: 1px solid var(--td-border-level-1-color);
  box-sizing: border-box;
}

.file-tabs {
  display: flex;
  min-width: 0;
  flex: 1;
  align-items: flex-end;
  gap: 6px;
  overflow: auto hidden;
}

.file-tab {
  display: inline-flex;
  max-width: 220px;
  height: 36px;
  align-items: center;
  gap: 8px;
  padding: 0 10px;
  border: 1px solid transparent;
  border-bottom: 0;
  border-radius: 11px 11px 0 0;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  font-size: 13px;
}

.file-tab span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-tab:hover,
.file-tab.is-active {
  border-color: var(--td-border-level-1-color);
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
}

.file-tab-close {
  display: inline-flex;
  width: 20px;
  height: 20px;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  padding: 0;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: inherit;
  cursor: pointer;
  opacity: 0.68;
}

.file-tab-close:hover {
  background: var(--td-bg-color-container-hover);
  opacity: 1;
}

.file-tab-add {
  display: inline-flex;
  width: 32px;
  height: 32px;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
}

.file-tab-add:hover {
  background: var(--td-bg-color-container-hover);
  color: var(--td-text-color-primary);
}

.file-workbench-toolbar {
  display: flex;
  min-height: 50px;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 10px 16px;
  border-bottom: 1px solid var(--td-border-level-1-color);
  background: var(--td-bg-color-container);
}

.file-breadcrumbs {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 5px;
  overflow: hidden;
  color: var(--td-text-color-placeholder);
  font-size: 13px;
}

.breadcrumb-item {
  overflow: hidden;
  max-width: 180px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.breadcrumb-item.is-current {
  color: var(--td-text-color-primary);
  font-weight: 700;
}

.file-toolbar-actions {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  gap: 8px;
}

.file-meta {
  color: var(--td-text-color-placeholder);
  font-size: 12px;
}

.file-workbench-main {
  display: grid;
  min-height: 0;
  flex: 1;
  grid-template-columns: 220px minmax(0, 1fr);
}

.file-workbench-info {
  display: flex;
  min-width: 0;
  padding: 18px 14px;
  border-right: 1px solid var(--td-border-level-1-color);
  background: color-mix(in srgb, var(--td-bg-color-container) 84%, transparent);
  flex-direction: column;
  gap: 10px;
}

.file-kind-badge {
  align-self: flex-start;
  padding: 4px 8px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--td-brand-color) 14%, transparent);
  color: var(--td-brand-color);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}

.file-info-name {
  color: var(--td-text-color-primary);
  font-size: 14px;
  font-weight: 700;
  word-break: break-all;
}

.file-info-path {
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  line-height: 1.5;
  word-break: break-all;
}

.file-preview-content {
  overflow: auto;
  min-width: 0;
  height: 100%;
  padding: 16px;
}

.file-preview-surface {
  min-height: 100%;
  border: 1px solid var(--td-border-level-1-color);
  border-radius: 14px;
  background: var(--td-bg-color-container);
  overflow: hidden;
}

.file-preview-fade-enter-active,
.file-preview-fade-leave-active {
  transition: opacity 0.16s ease, transform 0.16s ease;
}

.file-preview-fade-enter-from,
.file-preview-fade-leave-to {
  opacity: 0;
  transform: translateY(8px);
}

@media (max-width: 1180px) {
  .file-preview-workbench,
  .file-preview-workbench.has-browser {
    right: 24px;
    left: 24px;
  }

  .file-workbench-main {
    grid-template-columns: 1fr;
  }

  .file-workbench-info {
    display: none;
  }
}
</style>
