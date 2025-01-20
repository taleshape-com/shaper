import { redirect } from "@tanstack/react-router";
import { useAuth } from '../lib/auth'
import { useCallback } from "react";

export type QueryApiFunc = (url: string, options?: { method?: 'POST' | 'DELETE'; body?: any }) => Promise<any>;

export const useQueryApi = (): QueryApiFunc => {
  const auth = useAuth();
  return useCallback((async (url, options = {}) => {
    const jwt = await auth.getJwt();
    const response = await fetch(url, {
      headers: {
        "Content-Type": "application/json",
        Authorization: jwt,
      },
      method: options.method ?? 'GET',
      body: options.body ? JSON.stringify(options.body) : undefined,
    })
    if (response.status === 401) {
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
    if (response.status !== 200) {
      return response
        .json()
        .then((data: { error: string }) => {
          throw new Error(data.error);
        });
    }
    return response.json();
  }) as QueryApiFunc, [auth]);
};
