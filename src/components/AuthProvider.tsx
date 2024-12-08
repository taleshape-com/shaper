import { redirect } from '@tanstack/react-router'
import { useState } from 'react';
import {
  parseJwt,
  localStorageTokenKey,
  localStorageVariablesKey,
  AuthContext,
  Variables,
  zVariables,
} from '../lib/auth';


const getVariablesString = () => {
  return localStorage.getItem(localStorageVariablesKey) ?? '{}'
}
const getVariables = (s: string): Variables => {
  return zVariables.parse(JSON.parse(s))
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [jwt, setJwt] = useState<string | null>(null)
  const [variables, setVariables] = useState<Variables>(getVariables(getVariablesString()))
  const [hash, setHash] = useState<string>(getVariablesString())

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
      const res = await response.json()
      setJwt(res.jwt)
      return res.jwt;
    });
  }

  const goToLoginPage = () => {
    throw redirect({
      to: "/login",
      search: {
        // Use the current location to power a redirect after login
        // (Do not use `router.state.resolvedLocation` as it can
        // potentially lag behind the actual current location)
        redirect: location.pathname + location.search + location.hash,
      },
    });
  }

  const getJwt = async () => {
    if (jwt != null) {
      const claims = parseJwt(jwt)
      if (Date.now() / 1000 < claims.exp) {
        return jwt
      }
    }
    const token = localStorage.getItem(localStorageTokenKey)
    const vars = getVariables(getVariablesString())
    if (token == null) {
      goToLoginPage()
      return null
    }
    const newJwt = await refreshJwt(token, vars)
    if (newJwt == null) {
      goToLoginPage()
      return null
    }
    return newJwt
  }

  const login = async (token: string, vars?: Variables) => {
    const v = vars ?? getVariables(getVariablesString())
    const jwt = await refreshJwt(token, v)
    if (jwt != null) {
      localStorage.setItem(localStorageTokenKey, token)
      localStorage.setItem(localStorageVariablesKey, JSON.stringify(v))
      setHash(JSON.stringify(v))
      setVariables(v)
      return true
    }
    return false
  }

  const testLogin = async () => {
    const token = localStorage.getItem(localStorageTokenKey)
    if (token == null) {
      return false
    }
    const vars = getVariables(getVariablesString())
    const jwt = await refreshJwt(token, vars)
    return jwt != null
  }

  const updateVariables = async (text: string) => {
    try {
      const vars = getVariables(text)
      await login(localStorage.getItem(localStorageTokenKey) ?? "", vars)
      return true
    } catch {
      return false
    }
  }

  return (
    <AuthContext.Provider value={{ getJwt, login, testLogin, hash, variables, updateVariables }}>
      {children}
    </AuthContext.Provider>
  )
}


