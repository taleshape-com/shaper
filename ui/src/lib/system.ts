// SPDX-License-Identifier: MPL-2.0

import { localStorageJwtKey } from "./auth";

export interface ISystemConfig {
  loginRequired: boolean;
  workflowsEnabled: boolean;
}

export const localStorageSystemConfigKey = "shaper-system-config";

let systemConfig: ISystemConfig | null = null;

const configFromLocalStorage = () => {
  const storedVal = localStorage.getItem(localStorageSystemConfigKey);
  if (storedVal) {
    try {
      systemConfig = JSON.parse(storedVal) as ISystemConfig;
      if (typeof systemConfig !== "object" || systemConfig === null) {
        console.warn("Invalid system config format");
        return false;
      }
      return true;
    } catch (e) {
      console.warn("Invalid JSON in localStorage for system config", e);
    }
  }
  return false;
}

export const fetchSystemConfig = async () => {
  const response = await fetch(`${window.shaper.defaultBaseUrl}api/system/config`);
  if (!response.ok) {
    throw new Error("Failed to fetch system config");
  }
  const data = await response.json() as ISystemConfig;
  if (typeof data !== "object" || data === null) {
    throw new Error("Invalid system config format");
  }
  systemConfig = data;
  localStorage.setItem(localStorageSystemConfigKey, JSON.stringify(data));
}

const configChanged = (existingSystemConfig: ISystemConfig) => {
  const refreshedSystemConfig = getSystemConfig();
  return existingSystemConfig.loginRequired !== refreshedSystemConfig.loginRequired ||
    existingSystemConfig.workflowsEnabled !== refreshedSystemConfig.workflowsEnabled
}

export const reloadSystemConfig = async () => {
  const existingSystemConfig = getSystemConfig();
  await fetchSystemConfig();
  if (configChanged(existingSystemConfig)) {
    console.warn("System config has changed, reloading...");
    localStorage.removeItem(localStorageJwtKey);
    window.location.reload();
  }
}

export const loadSystemConfig = async () => {
  if (configFromLocalStorage()) {
    reloadSystemConfig();
    return;
  }
  await fetchSystemConfig();
}

export const getSystemConfig = (): ISystemConfig => {
  if (systemConfig === null) {
    throw new Error("System config not loaded");
  }
  return systemConfig;
}

