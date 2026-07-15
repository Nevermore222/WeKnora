import { defineStore } from 'pinia';
import type { PreviewFileRef } from '@/utils/filePreview';

function fileKey(file: PreviewFileRef) {
  return `${file.workspaceId || file.sessionId || file.source}:${file.relativePath || file.path}`;
}

export const useFilePreviewStore = defineStore('filePreview', {
  state: () => ({
    visible: false,
    browserVisible: false,
    current: null as PreviewFileRef | null,
    openedFiles: [] as PreviewFileRef[],
  }),
  getters: {
    currentKey(state) {
      return state.current ? fileKey(state.current) : '';
    },
  },
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
      const key = fileKey(file);
      const index = this.openedFiles.findIndex((item) => fileKey(item) === key);
      if (index >= 0) {
        this.openedFiles[index] = file;
      } else {
        this.openedFiles.push(file);
      }
      this.current = file;
      this.visible = true;
    },
    activate(file: PreviewFileRef) {
      this.current = file;
      this.visible = true;
    },
    closeFile(file: PreviewFileRef) {
      const key = fileKey(file);
      const index = this.openedFiles.findIndex((item) => fileKey(item) === key);
      if (index < 0) return;

      const closingCurrent = this.current ? fileKey(this.current) === key : false;
      this.openedFiles.splice(index, 1);
      if (!closingCurrent) return;

      const next = this.openedFiles[index] || this.openedFiles[index - 1] || null;
      this.current = next;
      this.visible = !!next;
    },
    close() {
      this.visible = false;
    },
    clear() {
      this.visible = false;
      this.current = null;
      this.openedFiles = [];
    },
  },
});
