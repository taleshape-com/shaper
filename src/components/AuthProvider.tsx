import { isEqual } from "lodash";
import { useCallback, useState } from "react";
import {
  parseJwt,
  localStorageTokenKey,
  localStorageVariablesKey,
  AuthContext,
  Variables,
  zVariables,
  localStorageJwtKey,
  localStorageLoginRequiredKey,
} from "../lib/auth";
import { goToLoginPage } from "../lib/utils";

const getVariablesString = () => {
  return localStorage.getItem(localStorageVariablesKey) ?? "{}";
};
const getVariables = (s: string): Variables => {
  return zVariables.parse(JSON.parse(s));
};

const getSessionToken = async (email: string, password: string) => {
  const response = await fetch("/api/login", {
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

const refreshJwt = async (token: string, vars: Variables) => {
  return fetch(`/api/auth/token`, {
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

const internalGetJwt = async (loginRequired: boolean) => {
  const jwt = localStorage.getItem(localStorageJwtKey);
  if (jwt != null) {
    const claims = parseJwt(jwt);
    if (Date.now() / 1000 < claims.exp) {
      return jwt;
    }
  }
  if (!loginRequired) {
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

const internalTestLogin = async (loginRequired: boolean) => {
  if (!loginRequired) {
    return true;
  }
  const token = localStorage.getItem(localStorageTokenKey);
  if (token == null || token === "") {
    return false;
  }
  const response = await fetch(`/api/auth/token`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ token }),
  });
  return response.status === 200;
};

export function AuthProvider({ children, initialLoginRequired }: { children: React.ReactNode, initialLoginRequired: boolean }) {
  const [loginRequired, setLoginRequired] = useState(initialLoginRequired);
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
      console.error(error)
      return false;
    }
  }, [updateJwtWithVars]);

  const getJwt = useCallback(async () => {
    return internalGetJwt(loginRequired)
  }, [loginRequired]);

  const testLogin = useCallback(async () => {
    return internalTestLogin(loginRequired)
  }, [loginRequired]);

  const handleSetLoginRequired = useCallback((isLoginRequired: boolean) => {
    localStorage.setItem(localStorageLoginRequiredKey, isLoginRequired ? "true" : "false")
    setLoginRequired(isLoginRequired);
  }, [])

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
        setLoginRequired: handleSetLoginRequired,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}
