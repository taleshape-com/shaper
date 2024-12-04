import { Dashboard } from './dashboard'
import { useState } from "react";
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
  return <div className="antialiased text-ctext dark:text-dtext">
    <Dashboard
      id={dashboardId}
      baseUrl={baseUrl}
      vars={vars}
      getJwt={() => getJwt({ baseUrl })}
      onVarsChanged={newVars => {
        setVars(newVars)
        if (onVarsChanged) {
          onVarsChanged(newVars)
        }
      }}
    />
  </div>
}

