// SPDX-License-Identifier: MPL-2.0

import { isEqual } from "lodash";
import { useCallback, useState } from "react";
import {
  localStorageTokenKey,
  localStorageVariablesKey,
  AuthContext,
  Variables,
  getVariables,
  getVariablesString,
  refreshJwt,
} from "../../lib/auth";

const getSessionToken = async (email: string, password: string) => {
  const response = await fetch(`${window.shaper.defaultBaseUrl}api/login`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ email, password }),
  });
  if (response.status !== 200) return null;
  const data = await response.json();
  return data.token;
};

export function AuthProvider ({ children }: { children: React.ReactNode }) {
  const [variables, setVariables] = useState<Variables>(
    getVariables(getVariablesString()),
  );
  const [hash, setHash] = useState<string>(getVariablesString());

  const updateJwtWithVars = useCallback(async (token: string, vars: Variables) => {
    const jwt = await refreshJwt(token, vars);
    if (!jwt) {
      return false;
    }
    if (token !== "") {
      localStorage.setItem(localStorageTokenKey, token);
    }
    localStorage.setItem(localStorageVariablesKey, JSON.stringify(vars));
    setHash(JSON.stringify(vars));
    setVariables(vars);
    return true;
  }, []);

  const login = useCallback(
    async (email: string, password: string, vars?: Variables) => {
      const sessionToken = await getSessionToken(email, password);
      if (!sessionToken) return false;
      const v = vars ?? getVariables(getVariablesString());
      return updateJwtWithVars(sessionToken, v);
    },
    [updateJwtWithVars],
  );

  const updateVariables = useCallback(async (text: string) => {
    try {
      const vars = getVariables(text);
      if (isEqual(vars, getVariables(getVariablesString()))) {
        return true;
      }
      const token = localStorage.getItem(localStorageTokenKey);
      return await updateJwtWithVars(token ?? "", vars);
    } catch (error) {
      console.error(error);
      return false;
    }
  }, [updateJwtWithVars]);

  return (
    <AuthContext.Provider
      value={{
        login,
        hash,
        variables,
        updateVariables,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}
