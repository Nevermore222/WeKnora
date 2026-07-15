<script setup lang="ts">
import { defineAsyncComponent, onBeforeUnmount, ref, shallowRef, watch } from 'vue';
import type { PreviewFileRef } from '@/utils/filePreview';
import { fetchWorkspaceFileBlob } from '@/api/workspace';

const VueOfficePptx = defineAsyncComponent(() => import('@vue-office/pptx'));

const props = defineProps<{
  file: PreviewFileRef;
}>();

const loading = ref(false);
const error = ref('');
const pptxData = shallowRef<ArrayBuffer | null>(null);
let loadId = 0;

async function loadPptx() {
  const currentLoad = ++loadId;
  error.value = '';
  pptxData.value = null;

  if (!props.file.workspaceId) {
    error.value = '当前文件缺少 workspaceId，无法预览';
    return;
  }

  loading.value = true;
  try {
    const blob = await fetchWorkspaceFileBlob(props.file.workspaceId, props.file.relativePath || props.file.path, 'raw');
    const data = await blob.arrayBuffer();
    if (currentLoad === loadId) {
      pptxData.value = data;
    }
  } catch (err: any) {
    if (currentLoad === loadId) {
      error.value = err?.message || 'PPTX 预览加载失败';
    }
  } finally {
    if (currentLoad === loadId) {
      loading.value = false;
    }
  }
}

watch(
  () => [props.file.workspaceId, props.file.relativePath, props.file.path],
  loadPptx,
  { immediate: true },
);

onBeforeUnmount(() => {
  loadId += 1;
  pptxData.value = null;
});
</script>

<template>
  <t-loading :loading="loading">
    <div v-if="pptxData" class="ppt-preview">
      <VueOfficePptx :src="pptxData" @error="(err: any) => { error = err?.message || 'PPTX 渲染失败'; pptxData = null; }" />
    </div>
    <t-empty v-else :description="error || '当前 PPTX 暂不可预览'" />
  </t-loading>
</template>

<style scoped>
.ppt-preview {
  min-height: calc(100vh - 245px);
  overflow: auto;
  padding: 16px;
  background: var(--td-bg-color-container);
}

:deep(.pptx-preview-wrapper),
:deep(.vue-office-pptx),
:deep(.vue-office-pptx-main) {
  width: 100%;
  min-height: calc(100vh - 285px);
  background: var(--td-bg-color-container);
}
</style>
