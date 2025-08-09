// SPDX-License-Identifier: MPL-2.0

const STORAGE_PREFIX = "shaper-dashboard-editor-";

export const editorStorage = {
  saveChanges(id: string, content: string) {
    localStorage.setItem(`${STORAGE_PREFIX}${id}`, content);
  },

  getChanges(id: string) {
    return localStorage.getItem(`${STORAGE_PREFIX}${id}`);
  },

  clearChanges(id: string) {
    localStorage.removeItem(`${STORAGE_PREFIX}${id}`);
  },

  hasUnsavedChanges(id: string) {
    return !!this.getChanges(id);
  },
};
