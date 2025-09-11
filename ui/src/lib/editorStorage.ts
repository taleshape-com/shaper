// SPDX-License-Identifier: MPL-2.0

const DASHBOARD_STORAGE_PREFIX = "shaper-dashboard-editor-";
const TASK_STORAGE_PREFIX = "shaper-task-editor-";

export const editorStorage = {
  saveChanges(id: string, content: string, t: 'dashboard' | 'task' = 'dashboard') {
    localStorage.setItem(`${t === 'task' ? TASK_STORAGE_PREFIX : DASHBOARD_STORAGE_PREFIX}${id}`, content);
  },

  getChanges(id: string, t: 'dashboard' | 'task' = 'dashboard') {
    return localStorage.getItem(`${t === 'task' ? TASK_STORAGE_PREFIX : DASHBOARD_STORAGE_PREFIX}${id}`);
  },

  clearChanges(id: string, t: 'dashboard' | 'task' = 'dashboard') {
    localStorage.removeItem(`${t === 'task' ? TASK_STORAGE_PREFIX : DASHBOARD_STORAGE_PREFIX}${id}`);
  },

  hasUnsavedChanges(id: string, t: 'dashboard' | 'task' = 'dashboard') {
    return !!this.getChanges(id, t);
  },
};
