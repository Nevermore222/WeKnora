<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { MessagePlugin } from 'tdesign-vue-next';
import { previewWorkspaceFile } from '@/api/workspace';
import type { PreviewFileRef } from '@/utils/filePreview';

const props = defineProps<{
  file: PreviewFileRef;
}>();

const loading = ref(false);
const content = ref('');

const lines = computed(() => {
  const value = content.value || '';
  return value.length ? value.split(/\r?\n/) : [''];
});

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
    <div class="text-preview-editor">
      <div v-for="(line, index) in lines" :key="index" class="text-preview-line">
        <span class="line-number">{{ index + 1 }}</span>
        <code>{{ line || ' ' }}</code>
      </div>
    </div>
  </t-loading>
</template>

<style scoped>
.text-preview-editor {
  min-height: calc(100vh - 245px);
  overflow: auto;
  padding: 14px 0;
  background:
    linear-gradient(90deg, color-mix(in srgb, var(--td-bg-color-container-hover) 72%, transparent) 0 64px, transparent 64px),
    var(--td-bg-color-container);
  color: var(--td-text-color-primary);
  font-family: "JetBrains Mono", "Cascadia Code", "SFMono-Regular", monospace;
  font-size: 13px;
  line-height: 1.72;
}

.text-preview-line {
  display: grid;
  min-width: max-content;
  grid-template-columns: 64px 1fr;
}

.line-number {
  padding-right: 14px;
  color: var(--td-text-color-placeholder);
  text-align: right;
  user-select: none;
}

.text-preview-line code {
  padding: 0 18px;
  color: inherit;
  font: inherit;
  white-space: pre;
}
</style>
