import { Dashboard } from './dashboard'
import { useState } from "react";
import { VarsParamSchema } from "../lib/utils";

export interface EmbedProps {
  baseUrl?: string;
  dashboardId: string;
  initialVars?: VarsParamSchema,
  getJwt: () => Promise<string>;
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
  return <div className="shaper-tailwind-scope">
    <Dashboard
      id={dashboardId}
      baseUrl={baseUrl}
      vars={vars}
      getJwt={getJwt}
      onVarsChanged={newVars => {
        setVars(newVars)
        if (onVarsChanged) {
          onVarsChanged(newVars)
        }
      }}
    />
  </div>
}

