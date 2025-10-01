// SPDX-License-Identifier: MPL-2.0

import { z } from "zod";
import { createFileRoute, isRedirect, Link } from "@tanstack/react-router";
import type { ErrorComponentProps } from "@tanstack/react-router";
import { RiPencilLine, RiFileCopyLine } from "@remixicon/react";
import { Dashboard } from "../components/dashboard";
import { Helmet } from "react-helmet";
import { useNavigate } from "@tanstack/react-router";
import {
  VarsParamSchema,
  varsParamSchema,
  copyToClipboard,
} from "../lib/utils";
import { useAuth, getJwt } from "../lib/auth";
import { useCallback, useState, useRef } from "react";
import { Result } from "../lib/types";
import { MenuProvider } from "../components/providers/MenuProvider";
import { MenuTrigger } from "../components/MenuTrigger";
import { VariablesMenu } from "../components/VariablesMenu";
import { PublicLink } from "../components/PublicLink";
import { useToast } from "../hooks/useToast";
import { Button } from "../components/tremor/Button";

export const Route = createFileRoute("/dashboards/$id")({
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
  const navigate = useNavigate({ from: "/dashboards/$id" });
  const [title, setTitle] = useState("Dashboard");
  const [visibility, setVisibility] = useState<Result["visibility"]>(undefined);
  const { toast } = useToast();

  // Ref for dashboard ID text selection
  const dashboardIdRef = useRef<HTMLElement>(null);

  const handleRedirectError = useCallback(
    (err: Error) => {
      if (isRedirect(err)) {
        navigate(err.options);
      }
    },
    [navigate],
  );

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

  const handleCopyDashboardId = async () => {
    const success = await copyToClipboard(params.id);
    if (success) {
      toast({
        title: "Dashboard ID copied",
        description: "Dashboard ID copied to clipboard",
      });
    } else {
      toast({
        title: "Error",
        description: "Failed to copy dashboard ID",
        variant: "error",
      });
    }
  };

  const handleDashboardIdClick = () => {
    if (dashboardIdRef.current) {
      const selection = window.getSelection();
      const range = document.createRange();
      range.selectNodeContents(dashboardIdRef.current);
      selection?.removeAllRanges();
      selection?.addRange(range);
    }
  };

  const MenuButton = (
    <MenuTrigger className="-ml-1 mt-0.5 py-[6px]" title={title}>
      <Link
        to="/dashboards/$id/edit"
        params={{ id: params.id }}
        search={() => ({ vars })}
        className="block px-4 py-3 hover:underline"
      >
        <RiPencilLine className="size-4 inline mr-1.5 mb-1" />
        Edit Dashboard
      </Link>
      <div className="mt-6 px-4">
        <div className="text-sm font-medium text-ctext2 dark:text-dtext2 mb-2">
          Dashboard ID
        </div>
        <div className="flex items-center space-x-2">
          <code
            ref={dashboardIdRef}
            onClick={handleDashboardIdClick}
            className="flex-grow px-2 py-1.5 bg-cbgs dark:bg-dbgs border border-cb dark:border-db rounded text-xs font-mono text-ctext dark:text-dtext overflow-hidden text-ellipsis whitespace-nowrap cursor-pointer hover:bg-cbga dark:hover:bg-dbga transition-colors"
          >
            {params.id}
          </code>
          <Button
            onClick={handleCopyDashboardId}
            variant="secondary"
            className="px-2 py-1.5 flex-shrink-0"
          >
            <RiFileCopyLine className="size-4" />
          </Button>
        </div>
      </div>
      <VariablesMenu />
      {(visibility === "public" || visibility === "password-protected") && (
        <div className="my-2 px-4">
          <PublicLink href={`../view/${params.id}`} />
        </div>
      )}
    </MenuTrigger>
  );

  const onDataChange = useCallback((data: Result) => {
    setTitle(data.name);
    setVisibility(data.visibility);
  }, []);

  return (
    <MenuProvider>
      <Helmet>
        <title>{title}</title>
        <meta name="description" content={title} />
      </Helmet>

      <div className="h-dvh">
        <Dashboard
          id={params.id}
          vars={vars}
          hash={auth.hash}
          getJwt={getJwt}
          menuButton={MenuButton}
          onVarsChanged={handleVarsChanged}
          onError={handleRedirectError}
          onDataChange={onDataChange}
        />
      </div>
    </MenuProvider>
  );
}
