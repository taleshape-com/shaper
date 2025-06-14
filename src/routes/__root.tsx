import { createRootRouteWithContext, Link, Outlet } from "@tanstack/react-router";
//import React from "react";
import { IAuthContext } from "../lib/auth";
import { QueryApiFunc } from "../hooks/useQueryApi";
import { Toaster } from "../components/tremor/Toaster";
import { DarkModeProvider } from "../components/DarkModeProvider";
import { loader } from "@monaco-editor/react";
import * as monaco from "monaco-editor";
import editorWorker from "monaco-editor/esm/vs/editor/editor.worker?worker";

// Initialize Monaco Editor from source files instead of CDN
self.MonacoEnvironment = {
  getWorker() {
    return new editorWorker();
  },
};
loader.config({ monaco });
loader.init();

//const TanStackRouterDevtools =
//process.env.NODE_ENV === "production"
//? () => null // Render nothing in production
//: React.lazy(() =>
//// Lazy load in development
//import("@tanstack/router-devtools").then((res) => ({
//default: res.TanStackRouterDevtools,
//// For Embedded Mode
//// default: res.TanStackRouterDevtoolsPanel
//})),
//);

interface RouterContext {
  auth: IAuthContext
  queryApi: QueryApiFunc
}

export const Route = createRootRouteWithContext<RouterContext>()({
  component: () => {
    return (
      <DarkModeProvider>
        <>
          <Toaster />
          <Outlet />
          {/* <TanStackRouterDevtools /> */}
        </>
      </DarkModeProvider>
    );
  },
  notFoundComponent: () => {
    return (
      <div>
        <p>Page not found</p>
        <Link to="/" className="underline">
          Go to homepage
        </Link>
      </div>
    );
  },
});
