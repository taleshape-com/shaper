const STORAGE_PREFIX = "shaper-dashboard-editor-";

export const editorStorage = {
  saveChanges(dashboardId: string, content: string) {
    localStorage.setItem(`${STORAGE_PREFIX}${dashboardId}`, content);
  },

  getChanges(dashboardId: string) {
    return localStorage.getItem(`${STORAGE_PREFIX}${dashboardId}`);
  },

  clearChanges(dashboardId: string) {
    localStorage.removeItem(`${STORAGE_PREFIX}${dashboardId}`);
  },

  hasUnsavedChanges(dashboardId: string) {
    return !!this.getChanges(dashboardId);
  },
};
