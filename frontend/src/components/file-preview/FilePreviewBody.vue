<script setup lang="ts">
import { computed } from 'vue';
import type { PreviewFileRef } from '@/utils/filePreview';
import { canInlinePreview, detectPreviewKind } from '@/utils/filePreview';
import TextPreview from './previews/TextPreview.vue';
import ImagePreview from './previews/ImagePreview.vue';
import PdfPreview from './previews/PdfPreview.vue';
import UnsupportedPreview from './previews/UnsupportedPreview.vue';

const props = defineProps<{
  file: PreviewFileRef;
}>();

const kind = computed(() => props.file.kind || detectPreviewKind(props.file.name, props.file.mimeType));
const canPreview = computed(() => canInlinePreview({ ...props.file, kind: kind.value }));
</script>

<template>
  <div class="file-preview-body">
    <TextPreview v-if="canPreview && (kind === 'markdown' || kind === 'text')" :file="file" />
    <ImagePreview v-else-if="canPreview && kind === 'image'" :file="file" />
    <PdfPreview v-else-if="canPreview && kind === 'pdf'" :file="file" />
    <UnsupportedPreview v-else :file="file" />
  </div>
</template>

<style scoped>
.file-preview-body {
  min-height: 360px;
  padding-top: 16px;
}
</style>
