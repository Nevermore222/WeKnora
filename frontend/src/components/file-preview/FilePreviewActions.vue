<script setup lang="ts">
import { MessagePlugin } from 'tdesign-vue-next';
import type { PreviewFileRef } from '@/utils/filePreview';
import { fileInfoText } from '@/utils/filePreview';

const props = defineProps<{
  file: PreviewFileRef;
}>();

async function copyText(text: string, successMessage: string) {
  try {
    await navigator.clipboard.writeText(text);
    MessagePlugin.success(successMessage);
  } catch {
    MessagePlugin.error('复制失败，请手动复制');
  }
}

function openDownload() {
  if (!props.file.downloadUrl) {
    MessagePlugin.warning('当前文件没有可下载地址');
    return;
  }
  window.open(props.file.downloadUrl, '_blank', 'noopener,noreferrer');
}
</script>

<template>
  <div class="file-preview-actions">
    <t-button size="small" theme="primary" variant="base" :disabled="!file.downloadUrl" @click="openDownload">
      下载
    </t-button>
    <t-button size="small" variant="outline" @click="copyText(file.relativePath || file.path, '已复制文件路径')">
      复制路径
    </t-button>
    <t-button size="small" variant="outline" @click="copyText(fileInfoText(file), '已复制文件信息')">
      复制信息
    </t-button>
    <t-tooltip content="浏览器安全限制下不能直接启动本机应用，请下载后选择打开方式">
      <t-button size="small" variant="text">
        打开方式
      </t-button>
    </t-tooltip>
  </div>
</template>

<style scoped>
.file-preview-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding: 12px 0 16px;
  border-bottom: 1px solid var(--td-border-level-1-color);
}
</style>
