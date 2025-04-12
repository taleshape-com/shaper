import "./index.css";
import ReactDOM from "react-dom/client";
import { EmbedComponent, EmbedProps } from "./components/Embed";

type EmbedArgs = EmbedProps & {
  container: HTMLElement;
};

export function dashboard({
  container,
  dashboardId,
  baseUrl,
  defaultVars,
  getJwt,
  onVarsChanged,
}: EmbedArgs) {
  container.classList.add("shaper-scope");

  ReactDOM.createRoot(container).render(
    <EmbedComponent
      dashboardId={dashboardId}
      baseUrl={baseUrl}
      defaultVars={defaultVars}
      getJwt={getJwt}
      onVarsChanged={onVarsChanged}
    />,
  );
}

// This alias is only exported for backward compatibility
export const embed = dashboard;
