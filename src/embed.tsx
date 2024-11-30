import "./index.css";
import ReactDOM from "react-dom/client";
import { Dashboard } from './components/dashboard'
import { VarsParamSchema } from "./lib/utils";
import { useState } from "react";

interface EmbedArgs {
  container: HTMLElement;
  baseUrl?: string;
  dashboardId: string;
  storeVarsInQueryParam?: string;
  initialVars?: VarsParamSchema,
  getJwt: () => Promise<string>;
  onVarsChanged?: (newVars: VarsParamSchema) => void;
}

function EmbedComponent({
  dashboardId,
  baseUrl,
  initialVars,
  getJwt,
  onVarsChanged,
}: Omit<EmbedArgs, 'container' | 'storeVarsInQueryParam'>) {
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

function storeVarsInQuery(param: string, vars: VarsParamSchema) {
  const url = new URL(window.location.toString())
  url.searchParams.set(param, encodeURIComponent(JSON.stringify(vars)))
  history.pushState(null, '', url);
}
function varsFromQuery(param: string) {
  const params = new URL(window.location.toString()).searchParams
  const p = params.get(param)
  if (!p) {
    return null
  }
  return JSON.parse(decodeURIComponent(p))
}

export function embed({
  container,
  dashboardId,
  baseUrl,
  storeVarsInQueryParam,
  initialVars,
  getJwt,
  onVarsChanged,
}: EmbedArgs) {
  if (storeVarsInQueryParam) {
    const fromQuery = varsFromQuery(storeVarsInQueryParam)
    if (fromQuery) {
      initialVars = fromQuery
    }
  }
  ReactDOM.createRoot(container).render(
    <EmbedComponent
      dashboardId={dashboardId}
      baseUrl={baseUrl}
      initialVars={initialVars}
      getJwt={getJwt}
      onVarsChanged={newVars => {
        if (storeVarsInQueryParam) {
          storeVarsInQuery(storeVarsInQueryParam, newVars)
        }
        if (onVarsChanged) {
          onVarsChanged(newVars)
        }
      }}
    />
  );
}

