// SPDX-License-Identifier: MPL-2.0

import "./index.css";
import { StrictMode } from "react";
import ReactDOM from "react-dom/client";
import { RemoveScroll } from "react-remove-scroll/UI";
import { createRouter } from "@tanstack/react-router";
import { AuthProvider } from "./components/providers/AuthProvider";
import { App } from "./App";

// Import the generated route tree
import { routeTree } from "./routeTree.gen";
import { ErrorComponent } from "@tanstack/react-router";
import { loadSystemConfig } from "./lib/system";
import "./lib/globals";

// Polyfill container queries
const supportsContainerQueries = "container" in document.documentElement.style;
if (!supportsContainerQueries) {
  // @ts-expect-error - This is a dynamic import
  import("https://cdn.skypack.dev/container-query-polyfill");
}

(RemoveScroll.defaultProps ?? {}).enabled = false;

// Create a new router instance
const router = createRouter({
  routeTree,
  basepath: window.shaper.defaultBaseUrl || "/",
  defaultPreload: false,
  defaultErrorComponent: ({ error }) => <ErrorComponent error={error} />,
  defaultStaleTime: 5000,
  context: {
    auth: undefined!, // This will be set after we wrap the app in an AuthProvider
    queryApi: undefined!, // This will be set after we wrap the app in an AuthProvider
  },
});

// Register the router instance for type safety
declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}

// Render the app
(async () => {
  const rootElement = document.getElementById("root")!;
  if (!rootElement.innerHTML) {
    await loadSystemConfig();
    const root = ReactDOM.createRoot(rootElement);
    root.render(
      <StrictMode>
        <AuthProvider>
          <App router={router} />
        </AuthProvider>
      </StrictMode>,
    );
  }
})();
