<script setup lang="ts">
import type { PreviewFileRef } from '@/utils/filePreview';
import { fileInfoText } from '@/utils/filePreview';

defineProps<{
  file: PreviewFileRef;
}>();
</script>

<template>
  <div class="unsupported-preview">
    <t-empty description="该文件暂不支持在线预览">
      <template #image>
        <div class="unsupported-icon">FILE</div>
      </template>
    </t-empty>
    <pre>{{ fileInfoText(file) }}</pre>
  </div>
</template>

<style scoped>
.unsupported-preview {
  display: grid;
  gap: 18px;
  padding: 24px;
  border: 1px dashed var(--td-border-level-2-color);
  border-radius: 12px;
  background: var(--td-bg-color-container-hover);
}

.unsupported-icon {
  display: grid;
  width: 64px;
  height: 64px;
  place-items: center;
  border-radius: 18px;
  background: var(--td-brand-color-light);
  color: var(--td-brand-color);
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.08em;
}

.unsupported-preview pre {
  margin: 0;
  padding: 12px;
  overflow: auto;
  border-radius: 8px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 1.6;
}
</style>
