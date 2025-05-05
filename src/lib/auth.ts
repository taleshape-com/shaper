import { z } from "zod";
import * as React from "react";
import { goToLoginPage } from "./utils";

export interface IAuthContext {
  getJwt: () => Promise<string>;
  login: (
    email: string,
    password: string,
    variables?: Variables,
  ) => Promise<boolean>;
  testLogin: () => Promise<boolean>;
  hash: string;
  loginRequired: boolean;
  setLoginRequired: (required: boolean) => void;
  variables: Variables;
  updateVariables: (text: string) => Promise<boolean>;
}

export const zVariables = z.record(
  z.string().min(1),
  z.union([z.string(), z.array(z.string())]),
);
export type Variables = (typeof zVariables)["_type"];

export const localStorageTokenKey = "shaper-session-token";
export const localStorageJwtKey = "shaper-jwt";
export const localStorageVariablesKey = "shaper-variables";
export const localStorageLoginRequiredKey = "shaper-login-required";

export const AuthContext = React.createContext<IAuthContext | null>(null);

export function useAuth() {
  const context = React.useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}

export async function logout() {
  const jwt = localStorage.getItem(localStorageJwtKey);
  if (jwt) {
    await fetch(`${window.shaper.defaultBaseUrl}api/logout`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: jwt,
      },
    })
  }
  localStorage.clear();
  return goToLoginPage();
}

export const checkLoginRequiredWithoutCache = async (): Promise<boolean> => {
  const response = await fetch(`${window.shaper.defaultBaseUrl}api/login/enabled`);
  if (!response.ok) {
    // Assume auth is required if we can't determine the status
    return true;
  }
  const data = await response.json() as { enabled: boolean };
  return data.enabled;
};

// Check if login is required using the auth status endpoint
export const checkLoginRequired = async (): Promise<boolean> => {
  const storedVal = localStorage.getItem(localStorageLoginRequiredKey)
  if (storedVal === "true") {
    return true;
  }
  if (storedVal === "false") {
    return false;
  }
  return checkLoginRequiredWithoutCache();
};

