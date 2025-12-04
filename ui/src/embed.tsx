// SPDX-License-Identifier: MPL-2.0

import "./index.css";
import ReactDOM from "react-dom/client";
import { EmbedComponent, type EmbedProps } from "./components/Embed";
import { RemoveScroll } from "react-remove-scroll/UI";
import "./lib/globals";

(RemoveScroll.defaultProps ?? {}).enabled = false;

// Function to inject custom CSS
function injectCustomCSS () {
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

type EmbedArgs = EmbedProps & {
  container: HTMLElement;
};

export function dashboard ({ container, ...initialProps }: EmbedArgs) {
  injectCustomCSS();
  container.classList.add("shaper-scope");

  // Expose renderMode on the global shaper object so the UI
  // (especially charts) can adjust behaviour for PDF rendering.
  if (typeof window !== "undefined") {
    // Ensure window.shaper exists
    // eslint-disable-next-line @typescript-eslint/ban-ts-comment
    // @ts-ignore
    window.shaper = window.shaper || {};
    // eslint-disable-next-line @typescript-eslint/ban-ts-comment
    // @ts-ignore
    window.shaper.renderMode = initialProps.renderMode ?? "interactive";
  }

  let updateProps: (newProps: Partial<EmbedProps>) => void = () => { };
  const updateSubscriber = (fn: typeof updateProps) => {
    updateProps = fn;
  };

  const root = ReactDOM.createRoot(container);
  root.render(
    <EmbedComponent initialProps={initialProps} updateSubscriber={updateSubscriber} />,
  );

  return {
    update: (newProps: Partial<EmbedProps>) => {
      updateProps(newProps);
    },

    destroy: () => {
      root.unmount();
      container.classList.remove("shaper-scope");
      updateProps = () => { };
    },
  };
}

// This alias is only exported for backward compatibility
export const embed = dashboard;
