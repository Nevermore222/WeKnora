<script setup lang="ts">
import { computed } from 'vue';
import { useFilePreviewStore } from '@/stores/filePreview';
import FilePreviewActions from './FilePreviewActions.vue';
import FilePreviewBody from './FilePreviewBody.vue';

const store = useFilePreviewStore();
const file = computed(() => store.current);
</script>

<template>
  <teleport to="body">
    <transition name="file-preview-fade">
      <section
        v-if="store.visible && file"
        class="file-preview-workbench"
        :class="{ 'has-browser': store.browserVisible }"
      >
        <header class="file-preview-header">
          <div class="file-preview-title-block">
            <div class="file-preview-title">{{ file.name }}</div>
            <div class="file-preview-meta">
              <span>{{ file.kind || 'file' }}</span>
              <span v-if="file.size != null">{{ file.size }} bytes</span>
              <span>{{ file.source }}</span>
            </div>
          </div>
          <t-button size="small" variant="text" shape="square" @click="store.close">
            <t-icon name="close" />
          </t-button>
        </header>
        <div class="file-preview-content">
          <FilePreviewActions :file="file" />
          <FilePreviewBody :file="file" />
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
  background: color-mix(in srgb, var(--td-bg-color-container) 96%, transparent);
  box-shadow: 0 24px 80px rgba(0, 0, 0, 0.28);
  flex-direction: column;
  backdrop-filter: blur(18px);
}

.file-preview-workbench.has-browser {
  right: 396px;
}

.file-preview-header {
  display: flex;
  min-height: 58px;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--td-border-level-1-color);
  box-sizing: border-box;
}

.file-preview-title-block {
  min-width: 0;
}

.file-preview-title {
  overflow: hidden;
  color: var(--td-text-color-primary);
  font-size: 15px;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-preview-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 4px;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
}

.file-preview-content {
  overflow: auto;
  height: 100%;
  padding: 0 16px 16px;
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
}
</style>
