import "./index.css";
import { StrictMode } from "react";
import ReactDOM from "react-dom/client";
import { createRouter } from "@tanstack/react-router";
import { AuthProvider } from './components/AuthProvider'
import { App } from "./App"

// Import the generated route tree
import { routeTree } from "./routeTree.gen";
import { ErrorComponent } from "@tanstack/react-router";

// Polyfill container queries
const supportsContainerQueries = "container" in document.documentElement.style;
if (!supportsContainerQueries) {
  // @ts-expect-error
  import("https://cdn.skypack.dev/container-query-polyfill");
}

// Create a new router instance
const router = createRouter({
  routeTree,
  defaultPreload: "intent",
  defaultErrorComponent: ({ error }) => <ErrorComponent error={error} />,
  defaultStaleTime: 5000,
  context: {
    auth: undefined!, // This will be set after we wrap the app in an AuthProvider
  },
});

// Register the router instance for type safety
declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}

// Render the app
const rootElement = document.getElementById("root")!;
if (!rootElement.innerHTML) {
  const root = ReactDOM.createRoot(rootElement);
  root.render(
    <StrictMode>
      <AuthProvider>
        <App router={router} />
      </AuthProvider>
    </StrictMode>,
  );
}
