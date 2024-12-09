import { z } from "zod";
import { createFileRoute, isRedirect, Link } from "@tanstack/react-router";
import type { ErrorComponentProps } from "@tanstack/react-router";
import { useDebouncedCallback } from "use-debounce";
import { RiCloseLargeLine, RiMenuLine } from "@remixicon/react";
import { Dashboard } from "../components/dashboard";
import { Helmet } from "react-helmet";
import { useNavigate } from "@tanstack/react-router";
import { cx, focusRing, hasErrorInput, varsParamSchema } from "../lib/utils";
import { useAuth } from "../lib/auth";
import { useState } from "react";
import { translate } from "../lib/translate";

export const Route = createFileRoute("/dashboards/$dashboardId")({
  validateSearch: z.object({
    vars: varsParamSchema,
  }),
  errorComponent: DashboardErrorComponent,
  notFoundComponent: () => {
    return (
      <div>
        <p>Dashboard not found</p>
        <Link to="/" className="underline">
          Go to homepage
        </Link>
      </div>
    );
  },
  component: DashboardViewComponent,
});

function DashboardErrorComponent({ error }: ErrorComponentProps) {
  return (
    <div className="p-4 m-4 bg-red-200 rounded-md">
      <p>{error.message}</p>
    </div>
  );
}
function DashboardViewComponent() {
  const { vars } = Route.useSearch();
  const params = Route.useParams();
  const auth = useAuth();
  const navigate = useNavigate({ from: "/dashboards/$dashboardId" });
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const [hasVariableError, setHasVariableError] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const handleRedirectError = (err: Error) => {
    if (isRedirect(err)) {
      navigate(err);
      return
    }
    setError(err)
  };

  const MenuButton = (
    <button
      className="px-1"
      onClick={() => setIsMenuOpen(true)}
    >
      <RiMenuLine className="py-1 size-7 text-ctext2 dark:text-dtext2 hover:text-ctext hover:dark:text-dtext transition-colors" />
    </button>
  );

  const onVariablesEdit = useDebouncedCallback((value) => {
    auth.updateVariables(value).then(
      (ok) => {
        setHasVariableError(!ok);
      },
      () => {
        setHasVariableError(true);
      },
    );
  }, 500);

  if (error) {
    return <DashboardErrorComponent error={error} reset={() => { }} />;
  }

  return (
    <>
      <Helmet>
        <title>{params.dashboardId}</title>
        <meta name="description" content={params.dashboardId} />
      </Helmet>
      <div
        className={cx("pb-8 pt-1", {
          "h-dvh sm:h-fit overflow-y-hidden sm:overflow-y-auto": isMenuOpen,
        })}
        onClick={() => {
          if (isMenuOpen) {
            setIsMenuOpen(false);
          }
        }}
      >
        <Dashboard
          id={params.dashboardId}
          vars={vars}
          hash={auth.hash}
          getJwt={auth.getJwt}
          menuButton={MenuButton}
          onVarsChanged={(newVars) => {
            navigate({
              search: (old: any) => ({
                ...old,
                vars: newVars,
              }),
            });
          }}
          onError={handleRedirectError}
        />
      </div>
      <div
        className={cx(
          "fixed top-0 h-dvh w-full sm:w-fit bg-cbga dark:bg-dbga shadow-xl ease-in-out delay-75 duration-300 z-40",
          {
            "-translate-x-[calc(100vw+50px)] ": !isMenuOpen,
          },
        )}
      >
        <button onClick={() => setIsMenuOpen(false)}>
          <RiCloseLargeLine className="pl-1 py-1 ml-2 mt-2 size-7 text-ctext2 dark:text-dtext2 hover:text-ctext hover:dark:text-dtext transition-colors" />
        </button>
        <div className="mt-6 px-5 w-full sm:w-96">
          <label>
            <span className="text-lg font-medium font-display ml-1 mb-2 block">
              {translate("Variables")}
            </span>
            <textarea
              className={cx(
                "w-full px-3 py-1.5 bg-cbg dark:bg-dbg text-sm border border-cb dark:border-db shadow-sm outline-none ring-0 rounded-md font-mono resize-none",
                focusRing,
                hasVariableError && hasErrorInput,
              )}
              onChange={(event) => {
                onVariablesEdit(event.target.value);
              }}
              defaultValue={JSON.stringify(auth.variables, null, 2)}
              rows={4}
            ></textarea>
          </label>
        </div>
      </div>
    </>
  );
}
