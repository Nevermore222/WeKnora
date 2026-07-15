<script setup lang="ts">
import { computed } from 'vue';
import { useFilePreviewStore } from '@/stores/filePreview';
import { normalizeChatArtifact } from '@/utils/filePreview';

const props = defineProps<{
  session?: Record<string, any>;
}>();

const filePreviewStore = useFilePreviewStore();

const artifacts = computed(() => {
  const session = props.session || {};
  const candidates = [
    session.artifacts,
    session.artifact ? [session.artifact] : null,
    session.files,
    session.output_files,
    session.attachments,
  ];
  const rows = candidates.flatMap((item) => Array.isArray(item) ? item : []);
  return rows
    .filter((item) => item && (item.relative_path || item.relativePath || item.path || item.name))
    .map((item) => ({
      ...item,
      workspace_id: item.workspace_id || item.workspaceId || session.workspace_id || session.workspace_binding?.workspace_id,
      session_id: item.session_id || item.sessionId || session.id,
    }));
});

function openArtifact(artifact: Record<string, any>) {
  filePreviewStore.open(normalizeChatArtifact(artifact));
}
</script>

<template>
  <div v-if="artifacts.length" class="chat-artifact-list">
    <div class="chat-artifact-list__title">生成文件</div>
    <button
      v-for="artifact in artifacts"
      :key="artifact.relative_path || artifact.relativePath || artifact.path || artifact.name"
      type="button"
      class="chat-artifact-list__item"
      @click="openArtifact(artifact)"
    >
      <t-icon name="file" size="15px" />
      <span class="chat-artifact-list__name">{{ artifact.name || artifact.relative_path || artifact.path }}</span>
      <span class="chat-artifact-list__action">预览</span>
    </button>
  </div>
</template>

<style scoped>
.chat-artifact-list {
  display: grid;
  gap: 6px;
  max-width: 520px;
  margin: 8px 0;
  padding: 10px;
  border: 1px solid var(--td-border-level-1-color);
  border-radius: 10px;
  background: var(--td-bg-color-container);
}

.chat-artifact-list__title {
  color: var(--td-text-color-secondary);
  font-size: 12px;
  font-weight: 600;
}

.chat-artifact-list__item {
  display: flex;
  align-items: center;
  gap: 7px;
  min-width: 0;
  padding: 7px 8px;
  border: 1px solid transparent;
  border-radius: 8px;
  background: var(--td-bg-color-container-hover);
  color: var(--td-text-color-secondary);
  cursor: pointer;
}

.chat-artifact-list__item:hover {
  border-color: var(--td-brand-color-light);
  color: var(--td-brand-color);
}

.chat-artifact-list__name {
  min-width: 0;
  flex: 1;
  overflow: hidden;
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chat-artifact-list__action {
  flex-shrink: 0;
  font-size: 12px;
}
</style>
