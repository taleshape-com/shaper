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
  hash: string;
  variables: Variables;
  updateVariables: (text: string) => Promise<boolean>;
}

const zVariables = z.record(
  z.string().min(1),
  z.union([z.string(), z.array(z.string())]),
);
export type Variables = (typeof zVariables)["_type"];

export const localStorageTokenKey = "shaper-session-token";
export const localStorageJwtKey = "shaper-jwt";
export const localStorageVariablesKey = "shaper-variables";

export const AuthContext = React.createContext<IAuthContext | null>(null);

export function useAuth () {
  const context = React.useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}

export async function logout () {
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
  return goToLoginPage();
}

export const getVariablesString = () => {
  return localStorage.getItem(localStorageVariablesKey) ?? "{}";
};
export const getVariables = (s: string): Variables => {
  return zVariables.parse(JSON.parse(s));
};

export const refreshJwt = async (token: string, vars: Variables) => {
  return fetch(`${window.shaper.defaultBaseUrl}api/auth/token`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
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

export const getJwt = async () => {
  const jwt = localStorage.getItem(localStorageJwtKey);
  if (jwt != null) {
    const claims = parseJwt(jwt);
    if (Date.now() / 1000 < claims.exp) {
      return jwt;
    }
  }
  if (!getSystemConfig().loginRequired) {
    const vars = getVariables(getVariablesString());
    return refreshJwt("", vars) ?? "";
  }
  const token = localStorage.getItem(localStorageTokenKey);
  if (token == null) {
    throw goToLoginPage();
  }
  const vars = getVariables(getVariablesString());
  const newJwt = await refreshJwt(token, vars);
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
  const token = localStorage.getItem(localStorageTokenKey);
  if (token == null || token === "") {
    return false;
  }
  const response = await fetch(`${window.shaper.defaultBaseUrl}api/auth/token`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ token }),
  });
  return response.status === 200;
};
