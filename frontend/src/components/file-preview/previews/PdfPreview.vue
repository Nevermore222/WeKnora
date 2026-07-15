<script setup lang="ts">
import type { PreviewFileRef } from '@/utils/filePreview';

defineProps<{
  file: PreviewFileRef;
}>();
</script>

<template>
  <iframe v-if="file.rawUrl" class="pdf-preview" :src="file.rawUrl" :title="file.name" />
  <t-empty v-else description="当前 PDF 缺少可访问地址" />
</template>

<style scoped>
.pdf-preview {
  width: 100%;
  height: calc(100vh - 220px);
  min-height: 460px;
  border: 1px solid var(--td-border-level-1-color);
  border-radius: 10px;
  background: var(--td-bg-color-container);
}
</style>
