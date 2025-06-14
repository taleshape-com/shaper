import { Dashboard } from './dashboard'
import { useCallback, useEffect, useState, useRef } from "react";
import { parseJwt, VarsParamSchema } from "../lib/utils";
import { DarkModeProvider } from "./DarkModeProvider";

export type EmbedProps = {
  baseUrl?: string;
  dashboardId: string;
  getJwt: (args: { baseUrl?: string }) => Promise<string>;
  vars?: VarsParamSchema;
  onVarsChanged?: (newVars: VarsParamSchema) => void;
}

export function EmbedComponent({
  initialProps,
  updateSubscriber,
}: {
  initialProps: EmbedProps;
  updateSubscriber: (updateFn: (props: Partial<EmbedProps>) => void) => void;
}) {
  const [props, setProps] = useState<EmbedProps>(initialProps);
  const jwtRef = useRef<string | null>(null);

  let baseUrl = props.baseUrl ?? window.shaper.defaultBaseUrl;
  if (baseUrl[0] !== "/") {
    baseUrl = "/" + baseUrl;
  }
  if (baseUrl[baseUrl.length - 1] !== "/") {
    baseUrl = baseUrl + "/";
  }
  const { onVarsChanged, getJwt } = props;

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

  const handleGetJwt = useCallback(async () => {
    if (jwtRef.current != null) {
      const claims = parseJwt(jwtRef.current);
      // Check if the JWT is still valid for at least 10 seconds
      if ((Date.now() / 1000) + 10 < claims.exp) {
        return jwtRef.current;
      }
    }
    const newJwt = await getJwt({ baseUrl });
    jwtRef.current = newJwt
    return newJwt;
  }, [baseUrl, getJwt]);

  return (
    <DarkModeProvider>
      <Dashboard
        id={props.dashboardId}
        baseUrl={baseUrl}
        vars={props.vars}
        getJwt={handleGetJwt}
        onVarsChanged={handleVarsChanged}
      />
    </DarkModeProvider>
  );
}

