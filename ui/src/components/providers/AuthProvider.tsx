// SPDX-License-Identifier: MPL-2.0

import { isEqual } from "lodash";
import { useCallback, useState, useEffect } from "react";
import {
  localStorageTokenKey,
  localStorageVariablesKey,
  localStorageJwtKey,
  AuthContext,
  Variables,
  getVariables,
  getVariablesString,
  refreshJwt,
  getJwt,
} from "../../lib/auth";
import { parseJwt } from "../../lib/utils";

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
  const [userName, setUserName] = useState<string>("");
  const [userId, setUserId] = useState<string>("");

  const updateUserInfoFromJwt = useCallback((jwt: string | null) => {
    if (!jwt) {
      setUserName("");
      setUserId("");
      return;
    }
    try {
      const decoded = parseJwt(jwt);
      setUserName(decoded.userName || "");
      setUserId(decoded.userId || "");
    } catch {
      setUserName("");
      setUserId("");
    }
  }, []);

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
    updateUserInfoFromJwt(jwt);
    return true;
  }, [updateUserInfoFromJwt]);

  const refreshUserName = useCallback(async () => {
    try {
      const jwt = await getJwt(true);
      updateUserInfoFromJwt(jwt);
    } catch (error) {
      console.error("Failed to refresh username:", error);
    }
  }, [updateUserInfoFromJwt]);

  useEffect(() => {
    const jwt = localStorage.getItem(localStorageJwtKey);
    updateUserInfoFromJwt(jwt);
  }, [updateUserInfoFromJwt]);

  const login = useCallback(
    async (email: string, password: string, vars?: Variables) => {
      const sessionToken = await getSessionToken(email, password);
      if (!sessionToken) return false;
      const v = vars ?? getVariables(getVariablesString());
      return updateJwtWithVars(sessionToken, v);
    },
    [updateJwtWithVars],
  );

  const loginWithToken = useCallback(
    async (token: string, vars?: Variables) => {
      const v = vars ?? getVariables(getVariablesString());
      return updateJwtWithVars(token, v);
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
        loginWithToken,
        hash,
        variables,
        updateVariables,
        userName,
        userId,
        refreshUserName,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}
