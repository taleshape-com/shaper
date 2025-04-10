import "./index.css";
import ReactDOM from "react-dom/client";
import { VarsParamSchema } from "./lib/utils";
import { EmbedComponent, EmbedProps } from "./components/Embed";

function storeVarsInQuery(param: string, vars: VarsParamSchema) {
  const url = new URL(window.location.toString());
  url.searchParams.set(param, encodeURIComponent(JSON.stringify(vars)));
  history.pushState(null, "", url);
}
function varsFromQuery(param: string) {
  const params = new URL(window.location.toString()).searchParams;
  const p = params.get(param);
  if (!p) {
    return null;
  }
  return JSON.parse(decodeURIComponent(p));
}

type EmbedArgs = EmbedProps & {
  container: HTMLElement;
  storeVarsInQueryParam?: string;
};

export function dashboard({
  container,
  dashboardId,
  baseUrl,
  storeVarsInQueryParam,
  defaultVars,
  getJwt,
  onVarsChanged,
}: EmbedArgs) {
  if (storeVarsInQueryParam) {
    const fromQuery = varsFromQuery(storeVarsInQueryParam);
    if (fromQuery) {
      defaultVars = fromQuery;
    }
  }
  container.classList.add("shaper-scope");

  ReactDOM.createRoot(container).render(
    <EmbedComponent
      dashboardId={dashboardId}
      baseUrl={baseUrl}
      defaultVars={defaultVars}
      getJwt={getJwt}
      onVarsChanged={(newVars) => {
        if (storeVarsInQueryParam) {
          storeVarsInQuery(storeVarsInQueryParam, newVars);
        }
        if (onVarsChanged) {
          onVarsChanged(newVars);
        }
      }}
    />,
  );
}

// This alias is only exported for backward compatibility
export const embed = dashboard;
