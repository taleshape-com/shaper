// SPDX-License-Identifier: MPL-2.0

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
import { IApp } from "../lib/types";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeaderCell,
  TableRoot,
  TableRow,
} from "../components/tremor/Table";
import {
  RiAddFill,
  RiLayoutFill,
  RiSortAsc,
  RiSortDesc,
  RiGlobalLine,
  RiCodeSSlashFill,
  RiFile3Fill,
  RiBarChart2Line,
  RiUserSharedLine,
} from "@remixicon/react";
import { translate } from "../lib/translate";
import { getSystemConfig } from "../lib/system";
import { useQueryApi } from "../hooks/useQueryApi";
import { MenuProvider } from "../components/providers/MenuProvider";
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
import { RelativeDate } from "../components/RelativeDate";
import { cx } from "../lib/utils";

type DashboardListResponse = {
  apps: IApp[];
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
    return queryApi(`apps?sort=${sort}&order=${order}`).then(
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
  const [deleteDialog, setDeleteDialog] = useState<IApp | null>(null);

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
        sort: field === "updated" ? undefined : field,
        order: field === "updated" && newOrder === "desc" ? undefined : newOrder,
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

  const handleDelete = async (app: IApp) => {
    try {
      await queryApi(`${app.type === 'dashboard' ? 'dashboards' : 'tasks'}/${app.id}`, {
        method: "DELETE",
      });
      router.invalidate();
      toast({
        title: translate("Success"),
        description: translate(app.type === "dashboard" ? "Dashboard deleted successfully" : "Task deleted successfully"),
      });
    } catch (err) {
      if (isRedirect(err)) {
        return navigate(err.options);
      }
      toast({
        title: translate("Error"),
        description: err instanceof Error ? err.message : translate("Unknown error"),
        variant: "error",
      });
    }
  };

  if (!data) {
    return <div className="p-2">Loading...</div>;
  }

  return (
    <MenuProvider isHome>
      <Helmet>
        <title>{translate("Home")}</title>
        <meta name="description" content="Show a list of all dashboards and tasks" />
      </Helmet>

      <div className="md:px-4 pb-4 min-h-dvh flex flex-col">
        <div className="flex px-4 md:px-0">
          <MenuTrigger className="pr-1.5 py-3 -ml-1.5" />
          <h1 className="text-2xl font-semibold font-display flex-grow text-right md:text-left pb-2 pt-2.5">
            <RiLayoutFill
              className="size-5 inline hidden md:inline mr-1 -mt-1"
              aria-hidden={true}
            />
            {translate("Overview")}
          </h1>
        </div>

        <div className="bg-cbgs dark:bg-dbgs rounded-md shadow flex-grow md:p-6">
          {data.apps.length === 0 ? (
            <div className="my-4 flex flex-col items-center justify-center flex-grow">
              <RiLayoutFill
                className="mx-auto size-9 fill-ctext dark:fill-dtext"
                aria-hidden={true}
              />
              <p className="mt-2 mb-3 font-medium text-ctext dark:text-dtext">
                {"Create a first dashboard"}
              </p>
              <Link
                to="/new"
              >
                <Button>
                  <RiAddFill className="-ml-1 mr-0.5 size-5 shrink-0" aria-hidden={true} />
                  {translate("New")}
                </Button>
              </Link>
            </div>
          ) : (
            <TableRoot>
              <Table>
                <TableHead>
                  <TableRow>
                    {getSystemConfig().tasksEnabled && (
                      <TableHeaderCell
                      >
                        <Tooltip
                          showArrow={false}
                          content={translate("Type")}
                        >
                          <RiFile3Fill
                            className="size-5 fill-ctext2 dark:fill-dtext2 inline -mt-1 cursor-default"
                            aria-hidden={true}
                          />
                        </Tooltip>
                      </TableHeaderCell>
                    )}
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
                  {data.apps.map((app) => (
                    <TableRow
                      key={app.id}
                      className="group transition-colors duration-200"
                    >
                      {getSystemConfig().tasksEnabled && (
                        <TableCell className="font-medium text-ctext dark:text-dtext !p-0 group-hover:underline">
                          <Link
                            to={app.type === 'dashboard' ? "/dashboards/$id" : "/tasks/$id"}
                            params={{ id: app.id }}
                            className="p-4 block"
                          >
                            <Tooltip
                              showArrow={false}
                              content={<span className="capitalize">{app.type}</span>}
                            >
                              {app.type === "dashboard" ? (
                                <RiBarChart2Line
                                  className="size-5 fill-ctext2 dark:fill-dtext2 inline -mt-1"
                                  aria-hidden={true}
                                />
                              ) : (
                                <RiCodeSSlashFill
                                  className="size-5 fill-ctext2 dark:fill-dtext2 inline -mt-1"
                                  aria-hidden={true}
                                />
                              )}
                            </Tooltip>
                          </Link>
                        </TableCell>
                      )}
                      <TableCell className="font-medium text-ctext dark:text-dtext !p-0">
                        <Link
                          to={app.type === 'dashboard' ? "/dashboards/$id" : "/tasks/$id"}
                          params={{ id: app.id }}
                          className="p-4 block"
                        >
                          <span className="group-hover:underline">{app.name}</span>
                          {app.type === 'task'
                            ?
                            app.taskInfo && (
                              !(app.taskInfo.lastRunSuccess ?? true)
                                ? (
                                  <RuntimeTooltip
                                    lastRunAt={app.taskInfo.lastRunAt}
                                    nextRunAt={app.taskInfo.nextRunAt}
                                  >
                                    <span className="bg-cerr dark:bg-derr text-ctexti dark:text-dtexti text-xs rounded p-1 ml-2 opacity-60 group-hover:opacity-100 transition-opacity duration-200">
                                      {translate("Task Error")}
                                    </span>
                                  </RuntimeTooltip>
                                )
                                : app.taskInfo.nextRunAt != null && (
                                  <RuntimeTooltip
                                    lastRunAt={app.taskInfo.lastRunAt}
                                  >
                                    <span className="bg-cprimary dark:bg-dprimary text-ctexti dark:text-dtexti text-xs rounded p-1 ml-2 opacity-60 group-hover:opacity-100 transition-opacity duration-200">
                                      {translate("Next Run")}: <RelativeDate refresh date={new Date(app.taskInfo.nextRunAt)} />
                                    </span>
                                  </RuntimeTooltip>
                                )
                            )
                            : app.visibility === "public" ? (
                              <Tooltip
                                showArrow={false}
                                content={translate("This dashboard is public")}
                              >
                                <RiGlobalLine className="size-4 inline-block ml-2 -mt-0.5 fill-ctext dark:fill-dtext" />
                              </Tooltip>
                            ) : app.visibility === "password-protected" && (
                              <Tooltip
                                showArrow={false}
                                content={"This dashboard has a share link protected with a password"}
                              >
                                <RiUserSharedLine className="size-4 inline-block ml-2 -mt-0.5 fill-ctext dark:fill-dtext" />
                              </Tooltip>
                            )
                          }
                        </Link>
                      </TableCell>
                      <TableCell className="hidden md:table-cell text-ctext2 dark:text-dtext2 p-0">
                        <Link
                          to={app.type === 'dashboard' ? "/dashboards/$id" : "/tasks/$id"}
                          params={{ id: app.id }}
                          className="block p-4"
                        >
                          <Tooltip
                            showArrow={false}
                            content={new Date(app.createdAt).toLocaleString()}
                          >
                            {new Date(app.createdAt).toLocaleDateString()}
                          </Tooltip>
                        </Link>
                      </TableCell>
                      <TableCell className="hidden md:table-cell text-ctext2 dark:text-dtext2 p-0">
                        <Link
                          to={app.type === 'dashboard' ? "/dashboards/$id" : "/tasks/$id"}
                          params={{ id: app.id }}
                          className="block p-4"
                        >
                          <Tooltip
                            showArrow={false}
                            content={new Date(app.updatedAt).toLocaleString()}
                          >
                            {new Date(app.updatedAt).toLocaleDateString()}
                          </Tooltip>
                        </Link>
                      </TableCell>
                      <TableCell className="hidden md:table-cell">
                        <div className="flex gap-4">
                          <Link
                            to={app.type === 'dashboard' ? "/dashboards/$id/edit" : "/tasks/$id"}
                            params={{ id: app.id }}
                            className={cx(
                              "text-ctext2 dark:text-dtext2 hover:text-ctext dark:hover:text-dtext",
                              "hover:underline transition-colors duration-200",
                            )}
                          >
                            {translate("Edit")}
                          </Link>
                          <button
                            onClick={() => {
                              setDeleteDialog(app);
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

        <Dialog open={deleteDialog !== null} onOpenChange={(open) => !open && setDeleteDialog(null)}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>{translate("Confirm Deletion")}</DialogTitle>
              <DialogDescription>
                {deleteDialog && translate(
                  deleteDialog.type === 'dashboard'
                    ? 'Are you sure you want to delete the dashboard "%%"?'
                    : 'Are you sure you want to delete the task "%%"?'
                ).replace(
                  "%%",
                  deleteDialog.name,
                )}
              </DialogDescription>
            </DialogHeader>
            <DialogFooter>
              <Button onClick={() => setDeleteDialog(null)} variant="secondary">
                {translate("Cancel")}
              </Button>
              <Button
                variant="destructive"
                onClick={() => {
                  if (deleteDialog) {
                    handleDelete(deleteDialog);
                    setDeleteDialog(null);
                  }
                }}
              >
                {translate("Delete")}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>
    </MenuProvider >
  );
}

function RuntimeTooltip({ lastRunAt, nextRunAt, children }: {
  lastRunAt?: number | string;
  nextRunAt?: number | string;
  children?: React.ReactNode;
}) {
  if (lastRunAt == null) return children;
  const tooltipContent = (
    <>
      {translate("Last Run")}: <RelativeDate refresh date={new Date(lastRunAt)} />
      {nextRunAt != null && (
        <>
          <br />
          {translate("Next Run")}: <RelativeDate refresh date={new Date(nextRunAt)} />
        </>
      )}
    </>
  )
  return (
    <Tooltip
      showArrow={false}
      content={tooltipContent}
    >
      {children}
    </Tooltip>
  );
}
