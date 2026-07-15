<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue';
import type { PreviewFileRef } from '@/utils/filePreview';
import { createPreviewObjectUrl } from '@/utils/filePreview';

const props = defineProps<{
  file: PreviewFileRef;
}>();

const loading = ref(false);
const error = ref('');
const objectUrl = ref('');
let revokeCurrent: (() => void) | undefined;
let loadId = 0;

function resetObjectUrl() {
  if (revokeCurrent) {
    revokeCurrent();
    revokeCurrent = undefined;
  }
  objectUrl.value = '';
}

async function loadImage() {
  const currentLoad = ++loadId;
  resetObjectUrl();
  error.value = '';
  loading.value = true;
  try {
    const result = await createPreviewObjectUrl(props.file, props.file.mimeType);
    if (currentLoad !== loadId) {
      result.revoke();
      return;
    }
    objectUrl.value = result.objectUrl;
    revokeCurrent = result.revoke;
  } catch (err: any) {
    if (currentLoad === loadId) {
      error.value = err?.message || '图片预览加载失败';
    }
  } finally {
    if (currentLoad === loadId) {
      loading.value = false;
    }
  }
}

watch(
  () => [props.file.workspaceId, props.file.relativePath, props.file.path],
  loadImage,
  { immediate: true },
);

onBeforeUnmount(() => {
  loadId += 1;
  resetObjectUrl();
});
</script>

<template>
  <t-loading :loading="loading">
    <div class="image-preview">
      <img v-if="objectUrl" :src="objectUrl" :alt="file.name" />
      <t-empty v-else :description="error || '当前图片暂不可预览'" />
    </div>
  </t-loading>
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
