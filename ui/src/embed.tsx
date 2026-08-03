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

function checkIsDynamicHeight (container: HTMLElement): boolean {
  const inlineHeight = container.style.height;
  if (inlineHeight === "auto") return true;
  if (
    inlineHeight &&
    inlineHeight !== "auto" &&
    inlineHeight !== "initial" &&
    inlineHeight !== "inherit" &&
    inlineHeight !== "unset"
  ) {
    return false;
  }

  const initialHeight = container.clientHeight;

  // Temporarily detach child nodes to test if container height is content-dependent
  const childNodes = Array.from(container.childNodes);
  childNodes.forEach(node => node.remove());

  const heightWithoutChildren = container.clientHeight;
  const computedHeightWithoutChildren = window.getComputedStyle(container).height;

  // Re-attach child nodes synchronously
  childNodes.forEach(node => container.appendChild(node));

  // If removing content changed the height, container height depends on content (dynamic height)
  if (initialHeight !== heightWithoutChildren) {
    return true;
  }

  // If container height without content is 0px or auto, it's dynamic height
  if (computedHeightWithoutChildren === "0px" || computedHeightWithoutChildren === "auto") {
    return true;
  }

  return false;
}

export function dashboard ({ container, ...initialProps }: EmbedArgs) {
  injectCustomCSS();
  container.classList.add("shaper-scope");

  if (typeof window !== "undefined") {
    // Automatically detect if the container has dynamic height (e.g. height: auto)
    // versus a fixed/explicit height (e.g. height: 600px, height: 100vh).
    // Dynamic height containers require container-type: inline-size so content can expand the container height.
    const isDynamicHeight = checkIsDynamicHeight(container);
    if (isDynamicHeight) {
      container.classList.add("shaper-auto-height");
    }

    // Expose renderMode on the global shaper object so the UI
    // (especially charts) can adjust behaviour for PDF rendering.
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
      container.classList.remove("shaper-scope", "shaper-auto-height");
      updateProps = () => { };
    },
  };
}

// This alias is only exported for backward compatibility
export const embed = dashboard;
