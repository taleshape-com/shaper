import z from "zod";
import { isRedirect, useRouter } from "@tanstack/react-router";
import {
  ErrorComponent,
  createFileRoute,
  Link,
  useNavigate,
} from "@tanstack/react-router";
import type { ErrorComponentProps } from "@tanstack/react-router";
import { Helmet } from "react-helmet";
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
import { RiAddFill, RiLayoutFill, RiSortAsc, RiSortDesc } from "@remixicon/react";
import { translate } from "../lib/translate";
import { useQueryApi } from "../hooks/useQueryApi";
import { Menu } from "../components/Menu";
import { useState } from "react";
import { cx } from "../lib/utils";
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
    context: { queryApi },
    deps: { sort = "updated", order = "desc" },
  }) => {
    return queryApi(`/api/dashboards?sort=${sort}&order=${order}`).then(
      (fetchedData: DashboardListResponse) => {
        return fetchedData;
      },
    );
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
  const queryApi = useQueryApi();
  const router = useRouter();
  const [isMenuOpen, setIsMenuOpen] = useState(false);

  const handleSort = (field: "name" | "created" | "updated") => {
    const newOrder =
      field === (sort ?? "updated")
        ? (order ?? "desc") === "asc"
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
    if (field !== (sort ?? "updated")) return null;
    return (order ?? "desc") === "asc" ? (
      <RiSortAsc className="inline size-4" />
    ) : (
      <RiSortDesc className="inline size-4" />
    );
  };

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
      await queryApi(`/api/dashboards/${dashboard.id}`, {
        method: "DELETE",
      });
      // Reload the page to refresh the list
      router.invalidate();
    } catch (err) {
      if (isRedirect(err)) {
        return navigate(err);
      }
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
    <div className={cx("flex-1 px-4 pb-4 overflow-auto", { "ml-72": isMenuOpen })}>
      <Helmet>
        <title>{translate("Home")}</title>
        <meta name="description" content="Show a list of all dashboards" />
      </Helmet>
      <div className={cx("mb-4 flex", { "-ml-2": !isMenuOpen })}>
        <Menu inline isHome onOpenChange={setIsMenuOpen} />
      </div>
      <div className="bg-cbgl dark:bg-dbgl rounded-lg shadow px-6 pt-4 pb-6">
        <h1 className="text-2xl font-semibold font-display mb-2">
          <RiLayoutFill
            className="mx-auto -mt-1 mr-1 size-6 text-ctext dark:text-dtext inline"
            aria-hidden={true}
          />
          {translate("Dashboards")}
        </h1>
        {data.dashboards.length === 0 ? (
          <div className="my-4 flex flex-col items-center justify-center min-h-[calc(100vh-8.65rem)]">
            <RiLayoutFill
              className="mx-auto size-9 text-ctext dark:text-dtext"
              aria-hidden={true}
            />
            <p className="mt-2 mb-3 font-medium text-ctext dark:text-dtext">
              No dashboards yet
            </p>
            <Link
              to="/dashboard/new"
            >
              <Button>
                <RiAddFill className="-ml-1 mr-0.5 size-5 shrink-0" aria-hidden={true} />
                {translate("New Dashboard")}
              </Button>
            </Link>
          </div>
        ) : (
          <TableRoot className=" min-h-[calc(100vh-7.15rem)]">
            <Table>
              <TableHead>
                <TableRow>
                  <TableHeaderCell
                    onClick={() => handleSort("name" as const)}
                    className="text-md text-ctext dark:text-dtext cursor-pointer hover:bg-cbg dark:hover:bg-dbga"
                  >
                    {translate("Name")} <SortIcon field="name" />
                  </TableHeaderCell>
                  <TableHeaderCell
                    className="text-md text-ctext dark:text-dtext hidden md:table-cell cursor-pointer hover:bg-cbg dark:hover:bg-dbga"
                    onClick={() => handleSort("created" as const)}
                  >
                    {translate("Created")} <SortIcon field="created" />
                  </TableHeaderCell>
                  <TableHeaderCell
                    className="text-md text-ctext dark:text-dtext hidden md:table-cell cursor-pointer hover:bg-cbg dark:hover:bg-dbga"
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
                    className="group hover:bg-cbg dark:hover:bg-dbga transition-colors duration-200"
                  >
                    <TableCell className="font-medium text-ctext dark:text-dtext p-0">
                      <Link
                        to="/dashboards/$dashboardId"
                        params={{ dashboardId: dashboard.id }}
                        className="block p-4"
                      >
                        {dashboard.name}
                      </Link>
                    </TableCell>
                    <TableCell className="hidden md:table-cell text-ctext2 dark:text-dtext2 p-0">
                      <Link
                        to="/dashboards/$dashboardId"
                        params={{ dashboardId: dashboard.id }}
                        className="block p-4"
                        title={new Date(dashboard.createdAt).toLocaleString()}
                      >
                        {new Date(dashboard.createdAt).toLocaleDateString()}
                      </Link>
                    </TableCell>
                    <TableCell className="hidden md:table-cell text-ctext2 dark:text-dtext2 p-0">
                      <Link
                        to="/dashboards/$dashboardId"
                        params={{ dashboardId: dashboard.id }}
                        className="block p-4"
                        title={new Date(dashboard.updatedAt).toLocaleString()}
                      >
                        {new Date(dashboard.updatedAt).toLocaleDateString()}
                      </Link>
                    </TableCell>
                    <TableCell className="hidden md:table-cell">
                      <div className="flex gap-4">
                        <Link
                          to="/dashboards/$dashboardId/edit"
                          params={{ dashboardId: dashboard.id }}
                          className=" text-ctext2 dark:text-dtext2 hover:text-ctext dark:hover:text-dtext hover:underline transition-colors duration-200"
                        >
                          {translate("Edit")}
                        </Link>
                        <button
                          onClick={() => {
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
    </div>
  );
}
