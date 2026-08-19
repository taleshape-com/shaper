// SPDX-License-Identifier: MPL-2.0

import { z } from "zod";
import * as React from "react";
import { goToLoginPage, parseJwt } from "./utils";
import {
  loadSystemConfig,
  getSystemConfig,
  reloadSystemConfig,
} from "./system";

export interface IAuthContext {
  login: (
    email: string,
    password: string,
    variables?: Variables,
  ) => Promise<boolean>;
  loginWithToken: (token: string, variables?: Variables) => Promise<boolean>;
  hash: string;
  variables: Variables;
  updateVariables: (text: string) => Promise<boolean>;
  userName: string;
  userId: string;
  refreshUserName: () => Promise<void>;
}

const zVariables = z.record(
  z.string().min(1),
  z.union([z.string(), z.array(z.string())]),
);
export type Variables = z.infer<typeof zVariables>;

export const localStorageTokenKey = "shaper-session-token";
export const localStorageJwtKey = "shaper-jwt";
export const localStorageVariablesKey = "shaper-variables";

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
    });
  }
  localStorage.clear();
  loadSystemConfig();
  return goToLoginPage(true);
}

export const getVariablesString = () => {
  return localStorage.getItem(localStorageVariablesKey) ?? "{}";
};
export const getVariables = (s: string): Variables => {
  return zVariables.parse(JSON.parse(s));
};

export const refreshJwt = async (token: string, vars: Variables) => {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };
  if (getSystemConfig().loginRequired) {
    const jwt = localStorage.getItem(localStorageJwtKey);
    if (jwt) {
      try {
        const claims = parseJwt(jwt);
        if (claims && claims.exp && Date.now() / 1000 < claims.exp) {
          headers["Authorization"] = jwt.startsWith("Bearer ") ? jwt : `Bearer ${jwt}`;
        }
      } catch {
        // Ignore invalid JWT format
      }
    }
  }
  return fetch(`${window.shaper.defaultBaseUrl}api/auth/token`, {
    method: "POST",
    headers,
    body: JSON.stringify({
      token: token === "" ? undefined : token,
      variables: Object.keys(vars).length > 0 ? vars : undefined,
    }),
  }).then(async (response) => {
    if (response.status !== 200) {
      return null;
    }
    const res = await response.json();
    localStorage.setItem(localStorageJwtKey, res.jwt);
    return res.jwt;
  });
};

export const getJwt = async (force = false) => {
  const jwt = localStorage.getItem(localStorageJwtKey);
  if (jwt != null && !force) {
    const claims = parseJwt(jwt);
    // Add 30s buffer to prevent race conditions where token expires
    // between client check and server validation
    if (claims && claims.exp && Date.now() / 1000 < claims.exp - 30) {
      return jwt;
    }
  }
  if (!getSystemConfig().loginRequired) {
    const vars = getVariables(getVariablesString());
    return (await refreshJwt("", vars)) ?? "";
  }
  const token = localStorage.getItem(localStorageTokenKey);
  const vars = getVariables(getVariablesString());
  const newJwt = await refreshJwt(token || "", vars);
  if (newJwt == null) {
    throw goToLoginPage();
  }
  return newJwt;
};

export const testLogin = async () => {
  await reloadSystemConfig();
  if (!getSystemConfig().loginRequired) {
    return true;
  }
  const jwt = localStorage.getItem(localStorageJwtKey);
  if (jwt != null && jwt !== "") {
    const claims = parseJwt(jwt);
    if (claims && claims.exp && Date.now() / 1000 < claims.exp - 30) {
      return true;
    }
  }
  const token = localStorage.getItem(localStorageTokenKey);
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  if (jwt) {
    headers["Authorization"] = jwt.startsWith("Bearer ") ? jwt : `Bearer ${jwt}`;
  }
  try {
    const response = await fetch(`${window.shaper.defaultBaseUrl}api/auth/token`, {
      method: "POST",
      headers,
      body: JSON.stringify({ token: token || undefined }),
    });
    if (response.status === 200) {
      const res = await response.json();
      if (res.jwt) {
        localStorage.setItem(localStorageJwtKey, res.jwt);
      }
      return true;
    }
  } catch {
    return false;
  }
  return false;
};
