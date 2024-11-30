import * as React from 'react'

export interface IAuthContext {
  getJwt: () => Promise<string>
  login: (token: string) => Promise<boolean>
  testLogin: () => Promise<boolean>
}

export const localStorageTokenKey = 'shaper-token'

export const AuthContext = React.createContext<IAuthContext | null>(null)

export function useAuth() {
  const context = React.useContext(AuthContext)
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}

export function parseJwt(token: string) {
  const base64Url = token.split('.')[1];
  const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/');
  const jsonPayload = decodeURIComponent(window.atob(base64).split('').map(function(c) {
    return '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2);
  }).join(''));

  return JSON.parse(jsonPayload);
}
