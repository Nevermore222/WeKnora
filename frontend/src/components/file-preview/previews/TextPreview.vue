<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { MessagePlugin } from 'tdesign-vue-next';
import { previewWorkspaceFile } from '@/api/workspace';
import type { PreviewFileRef } from '@/utils/filePreview';
import { sanitizeMarkdownHTML, safeMarkdownToHTML } from '@/utils/security';
import {
  createChatMarkdownRenderer,
  renderChatMarkdown,
} from '@/utils/chatMarkdownRenderer';

const props = defineProps<{
  file: PreviewFileRef;
}>();

const loading = ref(false);
const content = ref('');
const sourceMode = ref(false);
const markdownRenderer = createChatMarkdownRenderer();

const isMarkdown = computed(() => props.file.kind === 'markdown' || /\.md(?:own)?$/i.test(props.file.name));

const lines = computed(() => {
  const value = content.value || '';
  return value.length ? value.split(/\r?\n/) : [''];
});

const renderedMarkdown = computed(() => renderChatMarkdown(content.value, {
  renderer: markdownRenderer,
  escapeMarkdown: safeMarkdownToHTML,
  sanitizeHtml: sanitizeMarkdownHTML,
  streaming: false,
}));

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
watch(() => props.file.path, () => {
  sourceMode.value = false;
  loadPreview();
});
</script>

<template>
  <t-loading :loading="loading">
    <div v-if="isMarkdown" class="markdown-preview-shell">
      <div class="markdown-preview-toolbar">
        <span>{{ sourceMode ? 'Markdown 源码' : 'Markdown 预览' }}</span>
        <t-button size="small" variant="text" @click="sourceMode = !sourceMode">
          {{ sourceMode ? '预览' : '查看源码' }}
        </t-button>
      </div>
      <div v-if="!sourceMode" class="markdown-preview markdown-content" v-html="renderedMarkdown" />
      <div v-else class="text-preview-editor is-source">
        <div v-for="(line, index) in lines" :key="index" class="text-preview-line">
          <span class="line-number">{{ index + 1 }}</span>
          <code>{{ line || ' ' }}</code>
        </div>
      </div>
    </div>
    <div v-else class="text-preview-editor">
      <div v-for="(line, index) in lines" :key="index" class="text-preview-line">
        <span class="line-number">{{ index + 1 }}</span>
        <code>{{ line || ' ' }}</code>
      </div>
    </div>
  </t-loading>
</template>

<style lang="less" scoped>
@import '../../css/chat-markdown.less';

.markdown-preview-shell {
  min-height: calc(100vh - 245px);
  background: var(--td-bg-color-container);
}

.markdown-preview-toolbar {
  position: sticky;
  top: 0;
  z-index: 1;
  display: flex;
  min-height: 48px;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 16px;
  border-bottom: 1px solid var(--td-border-level-1-color);
  background: color-mix(in srgb, var(--td-bg-color-container) 94%, transparent);
  color: var(--td-text-color-secondary);
  font-size: 13px;
  backdrop-filter: blur(12px);
}

.markdown-preview {
  max-width: 980px;
  padding: 26px 32px 56px;
  color: var(--td-text-color-primary);
  font-size: 16px;
  .chat-markdown-typography();
}

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

.text-preview-editor.is-source {
  min-height: calc(100vh - 293px);
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
