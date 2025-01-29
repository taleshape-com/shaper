import { z } from "zod";
import { createFileRoute, isRedirect, Link } from "@tanstack/react-router";
import type { ErrorComponentProps } from "@tanstack/react-router";
import { useDebouncedCallback } from "use-debounce";
import { RiPencilLine } from "@remixicon/react";
import { Dashboard } from "../components/dashboard";
import { Helmet } from "react-helmet";
import { useNavigate } from "@tanstack/react-router";
import {
  cx,
  focusRing,
  hasErrorInput,
  VarsParamSchema,
  varsParamSchema,
} from "../lib/utils";
import { useAuth } from "../lib/auth";
import { useCallback, useState } from "react";
import { translate } from "../lib/translate";
import { Result } from "../lib/dashboard";
import { Menu } from "../components/Menu";

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
  const [hasVariableError, setHasVariableError] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const [title, setTitle] = useState("Dashboard");

  const handleRedirectError = useCallback(
    (err: Error) => {
      if (isRedirect(err)) {
        navigate(err);
        return;
      }
      setError(err);
    },
    [navigate],
  );
  const handleVarsChanged = useCallback(
    (newVars: VarsParamSchema) => {
      navigate({
        replace: true,
        search: (old) => ({
          ...old,
          vars: newVars,
        }),
      });
    },
    [navigate],
  );

  const MenuButton = (
    <Menu>
      <Link
        to="/dashboards/$dashboardId/edit"
        params={{ dashboardId: params.dashboardId }}
        search={() => ({ vars })}
        className="block px-4 py-4 hover:underline"
      >
        <RiPencilLine className="size-4 inline mr-2 mb-1" />
        {translate("Edit Dashboard")}
      </Link>
      <div className="mt-6 px-4 w-full">
        <label>
          <span className="text-lg font-medium font-display ml-1 mb-2 block">
            {translate("Variables")}
          </span>
          <textarea
            className={cx(
              "w-full px-3 py-1.5 bg-cbgl dark:bg-dbgl text-sm border border-cb dark:border-db shadow-sm outline-none ring-0 rounded-md font-mono resize-none",
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
    </Menu>
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

  const onDataChange = useCallback((data: Result) => {
    setTitle(data.name);
  }, [])

  if (error) {
    return <DashboardErrorComponent error={error} reset={() => { }} />;
  }

  return (
    <>
      <Helmet>
        <title>{title}</title>
        <meta name="description" content={title} />
      </Helmet>
      <Dashboard
        id={params.dashboardId}
        vars={vars}
        hash={auth.hash}
        getJwt={auth.getJwt}
        menuButton={MenuButton}
        onVarsChanged={handleVarsChanged}
        onError={handleRedirectError}
        onDataChange={onDataChange}
      />
    </>
  );
}
