<script setup lang="ts">
import { onMounted, ref, watch } from 'vue';
import { MessagePlugin } from 'tdesign-vue-next';
import { previewWorkspaceFile } from '@/api/workspace';
import type { PreviewFileRef } from '@/utils/filePreview';

const props = defineProps<{
  file: PreviewFileRef;
}>();

const loading = ref(false);
const content = ref('');

async function loadPreview() {
  if (!props.file.workspaceId) {
    content.value = '当前文件缺少 workspaceId，无法在线预览。';
    return;
  }
  loading.value = true;
  try {
    const response = await previewWorkspaceFile(props.file.workspaceId, props.file.relativePath || props.file.path);
    content.value = response.content || '';
  } catch (error: any) {
    content.value = '';
    MessagePlugin.error(error?.message || '加载文本预览失败');
  } finally {
    loading.value = false;
  }
}

onMounted(loadPreview);
watch(() => props.file.path, loadPreview);
</script>

<template>
  <t-loading :loading="loading">
    <pre class="text-preview">{{ content }}</pre>
  </t-loading>
</template>

<style scoped>
.text-preview {
  min-height: 320px;
  max-height: calc(100vh - 220px);
  margin: 0;
  padding: 16px;
  overflow: auto;
  border: 1px solid var(--td-border-level-1-color);
  border-radius: 10px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
  font-family: "JetBrains Mono", "Cascadia Code", monospace;
  font-size: 13px;
  line-height: 1.7;
  white-space: pre-wrap;
  word-break: break-word;
}
</style>
