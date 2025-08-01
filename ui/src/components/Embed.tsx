// SPDX-License-Identifier: MPL-2.0

import { Dashboard } from './dashboard'
import { useCallback, useEffect, useState, useRef } from "react";
import { parseJwt, VarsParamSchema } from "../lib/utils";
import { DarkModeProvider } from "./providers/DarkModeProvider";

export type EmbedProps = {
  baseUrl?: string;
  dashboardId: string;
  getJwt?: () => Promise<string>;
  vars?: VarsParamSchema;
  onVarsChanged?: (newVars: VarsParamSchema) => void;
  onTitleChanged?: (title: string) => void;
}

const getPublicJwt = async (baseUrl: string, dashboardId: string): Promise<string | null> => {
  return fetch(`${baseUrl}api/auth/public`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      dashboardId,
    }),
  }).then(async (response) => {
    if (response.status !== 200) {
      return null;
    }
    const res = await response.json();
    return res.jwt;
  });
}

const getVisibility = async (baseUrl: string, dashboardId: string): Promise<string> => {
  return fetch(`${baseUrl}api/public/${dashboardId}/status`, {
    method: "GET",
    headers: {
      "Content-Type": "application/json",
    },
  }).then(async (response) => {
    if (response.status !== 200) {
      return 'private';
    }
    const res = await response.json();
    return res.visibility ?? 'private';
  });
}

export function EmbedComponent({
  initialProps,
  updateSubscriber,
}: {
  initialProps: EmbedProps;
  updateSubscriber: (updateFn: (props: Partial<EmbedProps>) => void) => void;
}) {
  const [props, setProps] = useState<EmbedProps>(initialProps);
  const { onVarsChanged, onTitleChanged, getJwt } = props;
  const jwtRef = useRef<string | null>(null);

  let baseUrl = props.baseUrl ?? window.shaper.defaultBaseUrl;
  if (!baseUrl.startsWith('http://') && !baseUrl.startsWith('https://') && baseUrl[0] !== "/") {
    baseUrl = "/" + baseUrl;
  }
  if (baseUrl[baseUrl.length - 1] !== "/") {
    baseUrl = baseUrl + "/";
  }
  useEffect(() => {
    updateSubscriber((newProps: Partial<EmbedProps>) => {
      setProps(prevProps => ({ ...prevProps, ...newProps }));
    });
  }, [updateSubscriber]);

  const handleVarsChanged = useCallback((vars: VarsParamSchema) => {
    setProps(prevProps => ({ ...prevProps, vars }));
    if (onVarsChanged) {
      onVarsChanged(vars);
    }
  }, [onVarsChanged]);

  const handleDataChanged = useCallback(({ name }: { name: string }) => {
    if (onTitleChanged) {
      onTitleChanged(name);
    }
  }, [onTitleChanged]);

  const handleGetJwt = useCallback(async () => {
    if (jwtRef.current != null) {
      const claims = parseJwt(jwtRef.current);
      // Check if the JWT is still valid for at least 10 seconds
      if ((Date.now() / 1000) + 10 < claims.exp) {
        return jwtRef.current;
      }
    }
    if (!getJwt) {
      const visibility = await getVisibility(baseUrl, props.dashboardId);
      if (visibility === 'private') {
        throw new Error("Dashboard is not public");
      }
      const newJwt = await getPublicJwt(baseUrl, props.dashboardId);
      if (newJwt == null) {
        throw new Error("Failed to retrieve JWT for public dashboard");
      }
      jwtRef.current = newJwt
      return newJwt;
    }
    const newJwt = await getJwt();
    jwtRef.current = newJwt
    return newJwt;
  }, [baseUrl, getJwt, props.dashboardId]);

  return (
    <DarkModeProvider>
      <Dashboard
        id={props.dashboardId}
        baseUrl={baseUrl}
        vars={props.vars}
        getJwt={handleGetJwt}
        onVarsChanged={handleVarsChanged}
        onDataChange={handleDataChanged}
      />
    </DarkModeProvider>
  );
}
