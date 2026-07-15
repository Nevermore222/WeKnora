<script setup lang="ts">
import type { PreviewFileRef } from '@/utils/filePreview';

defineProps<{
  file: PreviewFileRef;
}>();
</script>

<template>
  <div class="image-preview">
    <img v-if="file.rawUrl" :src="file.rawUrl" :alt="file.name" />
    <t-empty v-else description="当前图片缺少可访问地址" />
  </div>
</template>

<style scoped>
.image-preview {
  display: grid;
  min-height: 360px;
  place-items: center;
  overflow: auto;
  border: 1px solid var(--td-border-level-1-color);
  border-radius: 10px;
  background:
    linear-gradient(45deg, rgba(0, 0, 0, 0.04) 25%, transparent 25%),
    linear-gradient(-45deg, rgba(0, 0, 0, 0.04) 25%, transparent 25%),
    linear-gradient(45deg, transparent 75%, rgba(0, 0, 0, 0.04) 75%),
    linear-gradient(-45deg, transparent 75%, rgba(0, 0, 0, 0.04) 75%);
  background-position: 0 0, 0 10px, 10px -10px, -10px 0;
  background-size: 20px 20px;
}

.image-preview img {
  max-width: 100%;
  max-height: calc(100vh - 230px);
  object-fit: contain;
}
</style>
