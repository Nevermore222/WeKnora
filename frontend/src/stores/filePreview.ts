import { defineStore } from 'pinia';
import type { PreviewFileRef } from '@/utils/filePreview';

export const useFilePreviewStore = defineStore('filePreview', {
  state: () => ({
    visible: false,
    browserVisible: false,
    current: null as PreviewFileRef | null,
  }),
  actions: {
    openBrowser() {
      this.browserVisible = true;
    },
    closeBrowser() {
      this.browserVisible = false;
    },
    toggleBrowser() {
      this.browserVisible = !this.browserVisible;
    },
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
