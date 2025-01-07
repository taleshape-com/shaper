import { z } from "zod";
import * as React from "react";

export interface IAuthContext {
  getJwt: () => Promise<string>;
  login: (token: string, variables?: Variables) => Promise<boolean>;
  testLogin: () => Promise<boolean>;
  hash: string;
  variables: Variables;
  updateVariables: (text: string) => Promise<boolean>;
}

export const zVariables = z.record(
  z.string(),
  z.union([z.string(), z.array(z.string())]),
);
export type Variables = (typeof zVariables)["_type"];

export const localStorageTokenKey = "shaper-token";
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

export function parseJwt(token: string) {
  const base64Url = token.split(".")[1];
  const base64 = base64Url.replace(/-/g, "+").replace(/_/g, "/");
  const jsonPayload = decodeURIComponent(
    window
      .atob(base64)
      .split("")
      .map(function (c) {
        return "%" + ("00" + c.charCodeAt(0).toString(16)).slice(-2);
      })
      .join(""),
  );

  return JSON.parse(jsonPayload);
}

export async function logout() {
  localStorage.clear();
}
