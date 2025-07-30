import { createRootRouteWithContext, Link, Outlet } from "@tanstack/react-router";
import { IAuthContext } from "../lib/auth";
import { QueryApiFunc } from "../hooks/useQueryApi";
import { Toaster } from "../components/tremor/Toaster";
import { DarkModeProvider } from "../components/providers/DarkModeProvider";

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
