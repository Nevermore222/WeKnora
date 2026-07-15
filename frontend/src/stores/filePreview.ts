import { defineStore } from 'pinia';
import type { PreviewFileRef } from '@/utils/filePreview';

export const useFilePreviewStore = defineStore('filePreview', {
  state: () => ({
    visible: false,
    current: null as PreviewFileRef | null,
  }),
  actions: {
    open(file: PreviewFileRef) {
      this.current = file;
      this.visible = true;
    },
    close() {
      this.visible = false;
    },
    clear() {
      this.visible = false;
      this.current = null;
    },
  },
});
