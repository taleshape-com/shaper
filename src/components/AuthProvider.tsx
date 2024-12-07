import { redirect } from '@tanstack/react-router'
import { useState } from 'react';
import {
  parseJwt,
  localStorageTokenKey,
  localStorageVariablesKey,
  AuthContext,
  Variables,
} from '../lib/auth';


const getVariablesString = () => {
  return localStorage.getItem(localStorageVariablesKey) ?? '{}'
}
const getVariables = (): Variables => {
  return JSON.parse(getVariablesString())
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [jwt, setJwt] = useState<string | null>(null)
  const [hash, setHash] = useState<string>(getVariablesString())

  const refreshJwt = async (token: string, variables: Variables) => {
    return fetch(`/api/login/token`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        token,
        variables: Object.keys(variables).length > 0 ? variables : undefined,
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
    const variables = getVariables()
    if (token == null) {
      goToLoginPage()
      return null
    }
    const newJwt = await refreshJwt(token, variables)
    if (newJwt == null) {
      goToLoginPage()
      return null
    }
    return newJwt
  }

  const login = async (token: string, variables: Variables) => {
    const jwt = await refreshJwt(token, variables)
    if (jwt != null) {
      localStorage.setItem(localStorageTokenKey, token)
      localStorage.setItem(localStorageVariablesKey, JSON.stringify(variables))
      setHash(JSON.stringify(variables))
      return true
    }
    return false
  }

  const testLogin = async () => {
    const token = localStorage.getItem(localStorageTokenKey)
    if (token == null) {
      return false
    }
    const variables = getVariables()
    const jwt = await refreshJwt(token, variables)
    return jwt != null
  }

  return (
    <AuthContext.Provider value={{ getJwt, login, testLogin, hash }}>
      {children}
    </AuthContext.Provider>
  )
}


