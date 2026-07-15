<script setup lang="ts">
import type { PreviewFileRef } from '@/utils/filePreview';
import { fileInfoText } from '@/utils/filePreview';

defineProps<{
  file: PreviewFileRef;
}>();
</script>

<template>
  <div class="unsupported-preview">
    <div class="unsupported-card">
      <div class="unsupported-icon">FILE</div>
      <div>
        <h3>暂不支持在线预览</h3>
        <p>该文件可以下载或复制路径后交给 OfficeCLI/skill 处理。后续可接入 preview-service 转换为 PDF/HTML。</p>
      </div>
    </div>
    <pre>{{ fileInfoText(file) }}</pre>
  </div>
</template>

<style scoped>
.unsupported-preview {
  display: grid;
  min-height: calc(100vh - 245px);
  align-content: start;
  gap: 18px;
  padding: 28px;
  background:
    radial-gradient(circle at 22% 16%, color-mix(in srgb, var(--td-brand-color) 12%, transparent), transparent 34%),
    var(--td-bg-color-container);
}

.unsupported-card {
  display: flex;
  align-items: center;
  gap: 18px;
  padding: 20px;
  border: 1px dashed var(--td-border-level-2-color);
  border-radius: 14px;
  background: color-mix(in srgb, var(--td-bg-color-container-hover) 78%, transparent);
}

.unsupported-icon {
  display: grid;
  width: 64px;
  height: 64px;
  flex-shrink: 0;
  place-items: center;
  border-radius: 18px;
  background: var(--td-brand-color-light);
  color: var(--td-brand-color);
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.08em;
}

.unsupported-card h3 {
  margin: 0 0 6px;
  color: var(--td-text-color-primary);
  font-size: 16px;
}

.unsupported-card p {
  margin: 0;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 1.7;
}

.unsupported-preview pre {
  margin: 0;
  padding: 14px;
  overflow: auto;
  border: 1px solid var(--td-border-level-1-color);
  border-radius: 10px;
  background: color-mix(in srgb, var(--td-bg-color-container-hover) 68%, transparent);
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 1.6;
}
</style>
