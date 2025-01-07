import z from "zod";
import { redirect, useRouter } from "@tanstack/react-router";
import {
  ErrorComponent,
  createFileRoute,
  Link,
  useNavigate,
} from "@tanstack/react-router";
import type { ErrorComponentProps } from "@tanstack/react-router";
import { Helmet } from "react-helmet";
import { logout, useAuth } from "../lib/auth";
import { IDashboard } from "../lib/dashboard";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeaderCell,
  TableRoot,
  TableRow,
} from "../components/tremor/Table";
import { RiSortAsc, RiSortDesc } from "@remixicon/react";
import { translate } from "../lib/translate";
import { Button } from "../components/tremor/Button";

type DashboardListResponse = {
  dashboards: IDashboard[];
};

export const Route = createFileRoute("/")({
  validateSearch: z.object({
    sort: z.enum(["name", "created", "updated"]).optional(),
    order: z.enum(["asc", "desc"]).optional(),
  }),
  loaderDeps: ({ search: { sort, order } }) => ({
    sort,
    order,
  }),
  loader: async ({
    context: { auth },
    deps: { sort = "name", order = "asc" },
  }) => {
    const jwt = await auth.getJwt();

    return fetch(`/api/dashboards?sort=${sort}&order=${order}`, {
      headers: {
        "Content-Type": "application/json",
        Authorization: jwt,
      },
    })
      .then(async (response) => {
        if (response.status === 401) {
          throw redirect({
            to: "/login",
            search: {
              // Use the current location to power a redirect after login
              // (Do not use `router.state.resolvedLocation` as it can
              // potentially lag behind the actual current location)
              redirect: location.pathname + location.search + location.hash,
            },
          });
        }
        if (response.status !== 200) {
          return response
            .json()
            .then((data: { Error: { Type: number; Msg: string } }) => {
              throw new Error(data.Error.Msg);
            });
        }
        return response.json();
      })
      .then((fetchedData: DashboardListResponse) => {
        return fetchedData;
      });
  },
  errorComponent: DashboardErrorComponent as any,
  component: Index,
});

function DashboardErrorComponent({ error }: ErrorComponentProps) {
  return <ErrorComponent error={error} />;
}

function Index() {
  const data = Route.useLoaderData();
  const { sort, order } = Route.useSearch();
  const navigate = useNavigate({ from: "/" });

  const handleSort = (field: "name" | "created" | "updated") => {
    const newOrder =
      field === (sort ?? "name")
        ? (order ?? "asc") === "asc"
          ? "desc"
          : "asc"
        : field === "name"
          ? "asc"
          : "desc";

    navigate({
      replace: true,
      search: (prev) => ({
        ...prev,
        sort: field === "name" ? undefined : field,
        order: field === "name" && newOrder === "asc" ? undefined : newOrder,
      }),
    });
  };

  const SortIcon = ({ field }: { field: "name" | "created" | "updated" }) => {
    if (field !== (sort ?? "name")) return null;
    return (order ?? "asc") === "asc" ? (
      <RiSortAsc className="inline size-4" />
    ) : (
      <RiSortDesc className="inline size-4" />
    );
  };
  const auth = useAuth();
  const router = useRouter();

  const handleDelete = async (dashboard: IDashboard) => {
    if (
      !window.confirm(
        translate(
          'Are you sure you want to delete the dashboard "%%"?',
        ).replace("%%", dashboard.name),
      )
    ) {
      return;
    }

    try {
      const jwt = await auth.getJwt();
      const response = await fetch(`/api/dashboards/${dashboard.id}`, {
        method: "DELETE",
        headers: {
          Authorization: jwt,
        },
      });

      if (!response.ok) {
        throw new Error("Failed to delete dashboard");
      }

      // Reload the page to refresh the list
      router.invalidate();
    } catch (err) {
      alert(
        "Error deleting dashboard: " +
        (err instanceof Error ? err.message : "Unknown error"),
      );
    }
  };

  if (!data) {
    return <div className="p-2">Loading dashboards...</div>;
  }

  return (
    <div className="p-4 max-w-[720px] mx-auto">
      <Helmet>
        <title>{translate("Overview")}</title>
        <meta name="description" content="Show a list of all dashboards" />
      </Helmet>
      <div className="mb-4 flex items-center">
        <h1 className="text-3xl font-semibold font-display flex-grow">
          {translate("Overview")}
        </h1>
        <Button
          asChild
          className="h-fit "
        >
          <Link
            to="/dashboard/new"
          >{translate("New")}</Link>
        </Button>
        <Button
          onClick={() => {
            logout();
            navigate({
              to: "/login",
              replace: true,
            });
          }}
          variant="secondary"
          className="h-fit ml-3"
        >
          {translate("Logout")}
        </Button>
      </div>
      {data.dashboards.length === 0 ? (
        <p>No dashboards yet</p>
      ) : (
        <TableRoot>
          <Table>
            <TableHead>
              <TableRow>
                <TableHeaderCell
                  onClick={() => handleSort("name" as const)}
                  className="text-md text-ctext dark:text-dtext cursor-pointer hover:bg-cbga dark:hover:bg-dbga"
                >
                  {translate("Name")} <SortIcon field="name" />
                </TableHeaderCell>
                <TableHeaderCell
                  className="text-md text-ctext dark:text-dtext hidden md:table-cell cursor-pointer hover:bg-cbga dark:hover:bg-dbga"
                  onClick={() => handleSort("created" as const)}
                >
                  {translate("Created")} <SortIcon field="created" />
                </TableHeaderCell>
                <TableHeaderCell
                  className="text-md text-ctext dark:text-dtext hidden md:table-cell cursor-pointer hover:bg-cbga dark:hover:bg-dbga"
                  onClick={() => handleSort("updated" as const)}
                >
                  {translate("Updated")} <SortIcon field="updated" />
                </TableHeaderCell>
                <TableHeaderCell className="text-md text-ctext dark:text-dtext hidden md:table-cell">
                  {translate("Actions")}
                </TableHeaderCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {data.dashboards.map((dashboard) => (
                <TableRow
                  key={dashboard.id}
                  className="group hover:bg-cbga dark:hover:bg-dbga cursor-pointer transition-colors duration-200"
                  onClick={() =>
                    navigate({
                      to: "/dashboards/$dashboardId",
                      params: { dashboardId: dashboard.id },
                    })
                  }
                >
                  <TableCell className="font-medium text-ctext dark:text-dtext">
                    {dashboard.name}
                  </TableCell>
                  <TableCell className="hidden md:table-cell text-ctext2 dark:text-dtext2">
                    <div title={new Date(dashboard.createdAt).toLocaleString()}>
                      {new Date(dashboard.createdAt).toLocaleDateString()}
                    </div>
                  </TableCell>
                  <TableCell className="hidden md:table-cell text-ctext2 dark:text-dtext2">
                    <div title={new Date(dashboard.updatedAt).toLocaleString()}>
                      {new Date(dashboard.updatedAt).toLocaleDateString()}
                    </div>
                  </TableCell>
                  <TableCell className="hidden md:table-cell">
                    <div className="flex gap-4">
                      <Link
                        to="/dashboards/$dashboardId/edit"
                        params={{ dashboardId: dashboard.id }}
                        className=" text-ctext2 dark:text-dtext2 hover:text-ctext dark:hover:text-dtext hover:underline transition-colors duration-200"
                        onClick={(e) => e.stopPropagation()}
                      >
                        {translate("Edit")}
                      </Link>
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          handleDelete(dashboard);
                        }}
                        className="text-cerr dark:text-derr opacity-90 hover:opacity-100 hover:underline"
                      >
                        {translate("Delete")}
                      </button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableRoot>
      )}
    </div>
  );
}
