import { useAuth } from '../lib/auth'
import { useCallback } from "react";
import { goToLoginPage } from "../lib/utils";

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
      throw goToLoginPage();
    }
    if (response.status !== 200 && response.status !== 201) {
      return response
        .json()
        .then((data: { error: string }) => {
          throw new Error(data.error);
        });
    }
    return response.json();
  }) as QueryApiFunc, [auth]);
};
