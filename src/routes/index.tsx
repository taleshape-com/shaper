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
import { MenuProvider } from "../components/MenuProvider";
import { MenuTrigger } from "../components/MenuTrigger";
import { Button } from "../components/tremor/Button";
import { Tooltip } from "../components/tremor/Tooltip";
import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../components/tremor/Dialog";
import { useToast } from "../hooks/useToast";

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
  const { toast } = useToast();
  const [deleteDashboardDialog, setDeleteDashboardDialog] = useState<IDashboard | null>(null);

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
    try {
      await queryApi(`/api/dashboards/${dashboard.id}`, {
        method: "DELETE",
      });
      // Reload the page to refresh the list
      router.invalidate();
      toast({
        title: translate("Success"),
        description: translate("Dashboard deleted successfully"),
      });
    } catch (err) {
      if (isRedirect(err)) {
        return navigate(err);
      }
      toast({
        title: translate("Error"),
        description: err instanceof Error ? err.message : translate("Unknown error"),
        variant: "error",
      });
    }
  };

  if (!data) {
    return <div className="p-2">Loading dashboards...</div>;
  }

  return (
    <MenuProvider isHome>
      <Helmet>
        <title>{translate("Home")}</title>
        <meta name="description" content="Show a list of all dashboards" />
      </Helmet>

      <div className="px-4 pb-4 min-h-dvh flex flex-col">
        <div className="flex">
          <MenuTrigger className="pr-1.5 py-3 -ml-1.5" />
          <h1 className="text-2xl font-semibold font-display flex-grow pb-2 pt-2.5">
            <RiLayoutFill
              className="size-5 inline mr-1 -mt-1"
              aria-hidden={true}
            />
            {translate("Dashboards")}
          </h1>
        </div>

        <div className="bg-cbgs dark:bg-dbgs rounded-lg shadow flex-grow px-6 pt-6">
          {data.dashboards.length === 0 ? (
            <div className="my-4 flex flex-col items-center justify-center flex-grow">
              <RiLayoutFill
                className="mx-auto size-9 text-ctext dark:text-dtext"
                aria-hidden={true}
              />
              <p className="mt-2 mb-3 font-medium text-ctext dark:text-dtext">
                No dashboards yet
              </p>
              <Link
                to="/new"
              >
                <Button>
                  <RiAddFill className="-ml-1 mr-0.5 size-5 shrink-0" aria-hidden={true} />
                  {translate("New Dashboard")}
                </Button>
              </Link>
            </div>
          ) : (
            <TableRoot>
              <Table>
                <TableHead>
                  <TableRow>
                    <TableHeaderCell
                      onClick={() => handleSort("name" as const)}
                      className="text-md text-ctext dark:text-dtext cursor-pointer hover:underline"
                    >
                      {translate("Name")} <SortIcon field="name" />
                    </TableHeaderCell>
                    <TableHeaderCell
                      className="text-md text-ctext dark:text-dtext hidden md:table-cell cursor-pointer hover:underline"
                      onClick={() => handleSort("created" as const)}
                    >
                      {translate("Created")} <SortIcon field="created" />
                    </TableHeaderCell>
                    <TableHeaderCell
                      className="text-md text-ctext dark:text-dtext hidden md:table-cell cursor-pointer hover:underline"
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
                      className="group transition-colors duration-200"
                    >
                      <TableCell className="font-medium text-ctext dark:text-dtext !p-0 group-hover:underline">
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
                        >
                          <Tooltip
                            showArrow={false}
                            content={new Date(dashboard.createdAt).toLocaleString()}
                          >
                            {new Date(dashboard.createdAt).toLocaleDateString()}
                          </Tooltip>
                        </Link>
                      </TableCell>
                      <TableCell className="hidden md:table-cell text-ctext2 dark:text-dtext2 p-0">
                        <Link
                          to="/dashboards/$dashboardId"
                          params={{ dashboardId: dashboard.id }}
                          className="block p-4"
                        >
                          <Tooltip
                            showArrow={false}
                            content={new Date(dashboard.updatedAt).toLocaleString()}
                          >
                            {new Date(dashboard.updatedAt).toLocaleDateString()}
                          </Tooltip>
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
                              setDeleteDashboardDialog(dashboard);
                            }}
                            className="text-cerr dark:text-derr hover:text-cerra dark:hover:text-derra hover:underline"
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

        <Dialog open={deleteDashboardDialog !== null} onOpenChange={(open) => !open && setDeleteDashboardDialog(null)}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>{translate("Confirm Deletion")}</DialogTitle>
              <DialogDescription>
                {deleteDashboardDialog && translate('Are you sure you want to delete the dashboard "%%"?').replace(
                  "%%",
                  deleteDashboardDialog.name,
                )}
              </DialogDescription>
            </DialogHeader>
            <DialogFooter>
              <Button onClick={() => setDeleteDashboardDialog(null)} variant="secondary">
                {translate("Cancel")}
              </Button>
              <Button
                variant="destructive"
                onClick={() => {
                  if (deleteDashboardDialog) {
                    handleDelete(deleteDashboardDialog);
                    setDeleteDashboardDialog(null);
                  }
                }}
              >
                {translate("Delete")}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>
    </MenuProvider>
  );
}
