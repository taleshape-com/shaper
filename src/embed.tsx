import "./index.css";
import ReactDOM from "react-dom/client";
import { VarsParamSchema } from "./lib/utils";
import { EmbedComponent, EmbedProps } from "./components/Embed";

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

type EmbedArgs = EmbedProps & {
  container: HTMLElement;
  storeVarsInQueryParam?: string;
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
  container.classList.add("shaper-scope")
  ReactDOM.createRoot(container).render(
    <EmbedComponent
      dashboardId={dashboardId}
      baseUrl={baseUrl ?? (window as any).shaper.defaultBaseUrl}
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

