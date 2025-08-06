// SPDX-License-Identifier: MPL-2.0

import { z } from "zod";
import { createFileRoute, isRedirect, Link } from "@tanstack/react-router";
import type { ErrorComponentProps } from "@tanstack/react-router";
import { RiExternalLinkLine, RiPencilLine } from "@remixicon/react";
import { Dashboard } from "../components/dashboard";
import { Helmet } from "react-helmet";
import { useNavigate } from "@tanstack/react-router";
import { VarsParamSchema, varsParamSchema } from "../lib/utils";
import { useAuth } from "../lib/auth";
import { useCallback, useState } from "react";
import { translate } from "../lib/translate";
import { Result } from "../lib/dashboard";
import { MenuProvider } from "../components/providers/MenuProvider";
import { MenuTrigger } from "../components/MenuTrigger";
import { VariablesMenu } from "../components/VariablesMenu";

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
  const [title, setTitle] = useState("Dashboard");
  const [visibility, setVisibility] = useState<Result["visibility"]>(undefined);

  const handleRedirectError = useCallback((err: Error) => {
    if (isRedirect(err)) {
      navigate(err.options);
    }
  }, [navigate]);

  const handleVarsChanged = useCallback(
    (newVars: VarsParamSchema) => {
      navigate({
        search: (old) => ({
          ...old,
          vars: newVars,
        }),
      });
    },
    [navigate],
  );


  const MenuButton = (
    <MenuTrigger className="-ml-1 mt-0.5 py-[6px]" title={title}>
      <Link
        to="/dashboards/$dashboardId/edit"
        params={{ dashboardId: params.dashboardId }}
        search={() => ({ vars })}
        className="block px-4 py-4 hover:underline"
      >
        <RiPencilLine className="size-4 inline mr-2 mb-1" />
        {translate("Edit Dashboard")}
      </Link>
      <VariablesMenu />
      {visibility === 'public' && (
        <a
          href={`../view/${params.dashboardId}`}
          target="_blank"
          className="py-4 px-6 text-sm text-ctext2 dark:text-dtext2 hover:text-ctext dark:hover:text-dtext underline transition-colors duration-200 block">
          {translate("Public Link")}
          <RiExternalLinkLine className="size-3.5 inline ml-1 -mt-1 fill-ctext2 dark:fill-dtext2" />
        </a>
      )}
    </MenuTrigger>
  );

  const onDataChange = useCallback((data: Result) => {
    setTitle(data.name);
    setVisibility(data.visibility);
  }, [])

  return (
    <MenuProvider>
      <Helmet>
        <title>{title}</title>
        <meta name="description" content={title} />
      </Helmet>

      <div className="h-dvh">
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
      </div>
    </MenuProvider>
  );
}
