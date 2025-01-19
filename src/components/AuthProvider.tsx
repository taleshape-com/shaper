import { redirect } from "@tanstack/react-router";
import { isEqual } from "lodash";
import { useCallback, useEffect, useState } from "react";
import {
  parseJwt,
  localStorageTokenKey,
  localStorageVariablesKey,
  AuthContext,
  Variables,
  zVariables,
  localStorageJwtKey,
} from "../lib/auth";

const getVariablesString = () => {
  return localStorage.getItem(localStorageVariablesKey) ?? "{}";
};
const getVariables = (s: string): Variables => {
  return zVariables.parse(JSON.parse(s));
};

const goToLoginPage = () => {
  return redirect({
    to: "/login",
    replace: true,
    search: {
      // Use the current location to power a redirect after login
      // (Do not use `router.state.resolvedLocation` as it can
      // potentially lag behind the actual current location)
      redirect: location.pathname + location.search + location.hash,
    },
  });
};

// Check if login is required using the auth status endpoint
const checkLoginRequired = async () => {
  const response = await fetch("/api/auth/enabled");
  if (!response.ok) {
    // Assume auth is required if we can't determine the status
    return true;
  }
  const data = await response.json();
  return data.enabled;
};

const refreshJwt = async (token: string, vars: Variables) => {
  return fetch(`/api/login/token`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      token,
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

const internalGetJwt = async (loginRequired: boolean) => {
  if (!loginRequired) {
    const vars = getVariables(getVariablesString());
    return refreshJwt("", vars) ?? "";
  }

  const jwt = localStorage.getItem(localStorageJwtKey);
  if (jwt != null) {
    const claims = parseJwt(jwt);
    if (Date.now() / 1000 < claims.exp) {
      return jwt;
    }
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

const internalTestLogin = async (loginRequired: boolean) => {
  if (!loginRequired) {
    return true;
  }
  const token = localStorage.getItem(localStorageTokenKey);
  if (token == null) {
    return false;
  }
  const vars = getVariables(getVariablesString());
  const jwt = await refreshJwt(token, vars);
  return jwt != null;
};

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [loginRequired, setLoginRequired] = useState<boolean>(false);
  const [variables, setVariables] = useState<Variables>(
    getVariables(getVariablesString()),
  );
  const [hash, setHash] = useState<string>(getVariablesString());

  // Check if login is required on mount
  useEffect(() => {
    checkLoginRequired().then((required) => {
      setLoginRequired(required);
    });
  }, []);

  const login = useCallback(
    async (token: string, vars?: Variables) => {
      const v = vars ?? getVariables(getVariablesString());
      const jwt = await refreshJwt(token, v);
      if (jwt != null) {
        if (loginRequired) {
          localStorage.setItem(localStorageTokenKey, token);
        }
        localStorage.setItem(localStorageVariablesKey, JSON.stringify(v));
        setHash(JSON.stringify(v));
        setVariables(v);
        return true;
      }
      return false;
    },
    [loginRequired],
  );

  const updateVariables = useCallback(
    async (text: string) => {
      try {
        const vars = getVariables(text);
        if (isEqual(vars, getVariables(getVariablesString()))) {
          return true;
        }
        return await login(
          localStorage.getItem(localStorageTokenKey) ?? "",
          vars,
        );
      } catch {
        return false;
      }
    },
    [login],
  );

  const getJwt = useCallback(
    () => internalGetJwt(loginRequired),
    [loginRequired],
  );

  const testLogin = useCallback(
    () => internalTestLogin(loginRequired),
    [loginRequired],
  );

  return (
    <AuthContext.Provider
      value={{
        getJwt,
        login,
        testLogin,
        hash,
        variables,
        updateVariables,
        loginRequired,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}
