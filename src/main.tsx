import "./index.css";
import { StrictMode } from "react";
import ReactDOM from "react-dom/client";
import { RouterProvider, createRouter } from "@tanstack/react-router";
import { AuthProvider, useAuth } from './auth'
import type { AuthContext } from "./auth";

// Import the generated route tree
import { routeTree } from "./routeTree.gen";
import { ErrorComponent } from "@tanstack/react-router";

// Create a new router instance
const router = createRouter({
  routeTree,
  defaultPreload: "intent",
  defaultErrorComponent: ({ error }) => <ErrorComponent error={error} />,
  defaultStaleTime: 5000,
  context: {
    auth: undefined! as AuthContext, // This will be set after we wrap the app in an AuthProvider
  },
});

// Register the router instance for type safety
declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}

function App() {
  const auth = useAuth()
  return <RouterProvider router={router} context={{ auth }} />
}

// Render the app
const rootElement = document.getElementById("root")!;
if (!rootElement.innerHTML) {
  const root = ReactDOM.createRoot(rootElement);
  root.render(
    <StrictMode>
      <AuthProvider>
        <App />
      </AuthProvider>
    </StrictMode>,
  );
}
