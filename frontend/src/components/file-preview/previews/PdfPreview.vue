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

async function loadPdf() {
  const currentLoad = ++loadId;
  resetObjectUrl();
  error.value = '';
  loading.value = true;
  try {
    const result = await createPreviewObjectUrl(props.file, 'application/pdf');
    if (currentLoad !== loadId) {
      result.revoke();
      return;
    }
    objectUrl.value = result.objectUrl;
    revokeCurrent = result.revoke;
  } catch (err: any) {
    if (currentLoad === loadId) {
      error.value = err?.message || 'PDF 预览加载失败';
    }
  } finally {
    if (currentLoad === loadId) {
      loading.value = false;
    }
  }
}

watch(
  () => [props.file.workspaceId, props.file.relativePath, props.file.path],
  loadPdf,
  { immediate: true },
);

onBeforeUnmount(() => {
  loadId += 1;
  resetObjectUrl();
});
</script>

<template>
  <t-loading :loading="loading">
    <iframe v-if="objectUrl" class="pdf-preview" :src="objectUrl" :title="file.name" />
    <t-empty v-else :description="error || '当前 PDF 暂不可预览'" />
  </t-loading>
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
