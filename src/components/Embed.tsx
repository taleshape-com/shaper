import { Dashboard } from './dashboard'
import { useCallback, useEffect, useState } from "react";
import { VarsParamSchema } from "../lib/utils";
import { RemoveScroll } from "react-remove-scroll/UI";

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

// Function to inject custom CSS
function injectCustomCSS() {
  if (window.shaper?.customCSS) {
    const existingStyles = document.head.getElementsByTagName("style");
    for (const style of existingStyles) {
      if (style.textContent === window.shaper.customCSS) {
        // Custom CSS already injected
        return;
      }
    }
    const styleElement = document.createElement("style");
    styleElement.textContent = window.shaper.customCSS;
    document.head.appendChild(styleElement);
  }
}

export type EmbedProps = {
  baseUrl?: string;
  dashboardId: string;
  getJwt: (args: { baseUrl?: string }) => Promise<string>;
} & (
    | {
      vars: VarsParamSchema;
      onVarsChanged: (newVars: VarsParamSchema) => void;
      defaultVars?: undefined;
    }
    | {
      vars?: undefined;
      onVarsChanged?: (newVars: VarsParamSchema) => void;
      defaultVars?: VarsParamSchema;
    }
  )


export function EmbedComponent({
  dashboardId,
  baseUrl = window.shaper.defaultBaseUrl,
  getJwt,
  vars,
  defaultVars,
  onVarsChanged,
}: EmbedProps) {
  const [manageStateInternally] = useState(!vars);
  const [internalVars, setInternalVars] = useState(defaultVars);

  const handleVarsChanged = useCallback((newVars: VarsParamSchema) => {
    if (manageStateInternally) {
      setInternalVars(newVars);
    }
    if (onVarsChanged) {
      onVarsChanged(newVars);
    }
  }, [onVarsChanged, manageStateInternally]);

  const handleGetJwt = useCallback(() => {
    return getJwt({ baseUrl });
  }, [baseUrl, getJwt]);

  // Inject custom CSS
  useEffect(() => {
    injectCustomCSS();
  });

  return <Dashboard
    id={dashboardId}
    baseUrl={baseUrl}
    vars={vars ?? internalVars}
    getJwt={handleGetJwt}
    onVarsChanged={handleVarsChanged}
  />;
}

