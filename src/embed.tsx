import "./index.css";
import ReactDOM from "react-dom/client";
import { RemoveScroll } from "react-remove-scroll/UI";
import { VarsParamSchema } from "./lib/utils";
import { EmbedComponent, EmbedProps } from "./components/Embed";

// Add type definition for the global shaper object
declare global {
  interface Window {
    shaper: {
      defaultBaseUrl: string;
      customCSS?: string;
    };
  }
}

(RemoveScroll.defaultProps ?? {}).enabled = false;

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

// Function to inject custom CSS
function injectCustomCSS() {
  if (window.shaper?.customCSS) {
    const styleElement = document.createElement("style");
    styleElement.textContent = window.shaper.customCSS;
    document.head.appendChild(styleElement);
  }
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
    const fromQuery = varsFromQuery(storeVarsInQueryParam);
    if (fromQuery) {
      initialVars = fromQuery;
    }
  }
  container.classList.add("shaper-scope");

  // Inject custom CSS before rendering
  injectCustomCSS();

  ReactDOM.createRoot(container).render(
    <EmbedComponent
      dashboardId={dashboardId}
      baseUrl={baseUrl ?? (window as any).shaper.defaultBaseUrl}
      initialVars={initialVars}
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
