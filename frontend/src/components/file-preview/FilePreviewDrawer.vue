<script setup lang="ts">
import { computed } from 'vue';
import { useFilePreviewStore } from '@/stores/filePreview';
import FilePreviewActions from './FilePreviewActions.vue';
import FilePreviewBody from './FilePreviewBody.vue';

const store = useFilePreviewStore();
const file = computed(() => store.current);
</script>

<template>
  <t-drawer
    :visible="store.visible"
    placement="right"
    size="620px"
    :footer="false"
    destroy-on-close
    @close="store.close"
  >
    <template #header>
      <div v-if="file" class="file-preview-header">
        <div class="file-preview-title">{{ file.name }}</div>
        <div class="file-preview-meta">
          <span>{{ file.kind || 'file' }}</span>
          <span v-if="file.size != null">{{ file.size }} bytes</span>
          <span>{{ file.source }}</span>
        </div>
      </div>
    </template>
    <div v-if="file" class="file-preview-drawer">
      <FilePreviewActions :file="file" />
      <FilePreviewBody :file="file" />
    </div>
  </t-drawer>
</template>

<style scoped>
.file-preview-header {
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

.file-preview-drawer {
  height: 100%;
}
</style>
