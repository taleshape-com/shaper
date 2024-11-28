import { redirect } from '@tanstack/react-router'
import * as React from 'react'

export interface AuthContext {
  getJwt: () => Promise<string>
  login: (token: string) => Promise<boolean>
  testLogin: () => Promise<boolean>
}

const localStorageTokenKey = 'shaper-token'

const AuthContext = React.createContext<AuthContext | null>(null)

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [jwt, setJwt] = React.useState<string | null>(null)

  const refreshJwt = async (token: string) => {
    return fetch(`/api/login/token`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ token }),
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
    if (token == null) {
      goToLoginPage()
      return null
    }
    const newJwt = await refreshJwt(token)
    if (newJwt == null) {
      goToLoginPage()
      return null
    }
    return newJwt
  }

  const login = async (token: string) => {
    const jwt = await refreshJwt(token)
    if (jwt != null) {
      localStorage.setItem(localStorageTokenKey, token)
      return true
    }
    return false
  }

  const testLogin = async () => {
    const token = localStorage.getItem(localStorageTokenKey)
    if (token == null) {
      return false
    }
    const jwt = await refreshJwt(token)
    return jwt != null
  }

  return (
    <AuthContext.Provider value={{ getJwt, login, testLogin }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const context = React.useContext(AuthContext)
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}

function parseJwt(token: string) {
  var base64Url = token.split('.')[1];
  var base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/');
  var jsonPayload = decodeURIComponent(window.atob(base64).split('').map(function(c) {
    return '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2);
  }).join(''));

  return JSON.parse(jsonPayload);
}
