import "./index.css";
import ReactDOM from "react-dom/client";
import { EmbedComponent, type EmbedProps } from "./components/Embed";
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

type EmbedArgs = EmbedProps & {
  container: HTMLElement;
};

export function dashboard({ container, ...initialProps }: EmbedArgs) {
  injectCustomCSS();
  container.classList.add("shaper-scope");

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
