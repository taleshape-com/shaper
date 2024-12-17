import { Dashboard } from './dashboard'
import { useCallback, useState } from "react";
import { VarsParamSchema } from "../lib/utils";

export interface EmbedProps {
  baseUrl?: string;
  dashboardId: string;
  initialVars?: VarsParamSchema,
  getJwt: (args: { baseUrl?: string }) => Promise<string>;
  onVarsChanged?: (newVars: VarsParamSchema) => void;
}


export function EmbedComponent({
  dashboardId,
  baseUrl,
  initialVars,
  getJwt,
  onVarsChanged,
}: EmbedProps) {
  const [vars, setVars] = useState(initialVars)
  const handleVarsChanged = useCallback((newVars: VarsParamSchema) => {
    setVars(newVars)
    if (onVarsChanged) {
      onVarsChanged(newVars)
    }
  }, [onVarsChanged]);
  const handleGetJwt = useCallback(() => {
    return getJwt({ baseUrl })
  }, [baseUrl, getJwt]);

  return <div className="antialiased text-ctext dark:text-dtext">
    <Dashboard
      id={dashboardId}
      baseUrl={baseUrl}
      vars={vars}
      getJwt={handleGetJwt}
      onVarsChanged={handleVarsChanged}
    />
  </div>
}

