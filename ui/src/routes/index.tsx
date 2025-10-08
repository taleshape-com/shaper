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
  RiGlobalLine,
  RiCodeSSlashFill,
  RiBarChart2Line,
  RiUserSharedLine,
  RiFolderAddFill,
  RiFolderFill,
  RiArrowRightSLine,
  RiDeleteBinLine,
  RiPencilLine,
  RiArrowDownSLine,
  RiArrowUpSLine,
  RiFolderAddLine,
} from "@remixicon/react";

import { useQueryApi } from "../hooks/useQueryApi";
import { MenuProvider } from "../components/providers/MenuProvider";
import { MenuTrigger } from "../components/MenuTrigger";
import { Button } from "../components/tremor/Button";
import { Tooltip } from "../components/tremor/Tooltip";
import { Input } from "../components/tremor/Input";
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
    path: z.string().optional(),
  }),
  loaderDeps: ({ search: { sort, order, path } }) => ({
    sort,
    order,
    path,
  }),
  loader: async ({
    context: { queryApi },
    deps: { sort = "name", order = "asc", path },
  }) => {
    const params = new URLSearchParams();
    if (sort) params.set("sort", sort);
    if (order) params.set("order", order);
    if (path) params.set("path", path);

    return queryApi(`apps?${params.toString()}`).then(
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
  const { sort, order, path = "/" } = Route.useSearch();
  const navigate = useNavigate({ from: "/" });
  const queryApi = useQueryApi();
  const router = useRouter();
  const { toast } = useToast();
  const [deleteDialog, setDeleteDialog] = useState<IApp | null>(null);
  const [folderDialog, setFolderDialog] = useState(false);
  const [folderName, setFolderName] = useState("");
  const [renameDialog, setRenameDialog] = useState<IApp | null>(null);
  const [renameName, setRenameName] = useState("");
  const [draggedItem, setDraggedItem] = useState<IApp | null>(null);
  const [dragOverTarget, setDragOverTarget] = useState<string | null>(null);

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
      <RiArrowDownSLine className="inline size-4" />
    ) : (
      <RiArrowUpSLine className="inline size-4 -mt-1" />
    );
  };

  const handleDelete = async (app: IApp) => {
    try {
      const endpoint =
        app.type === "dashboard"
          ? `dashboards/${app.id}`
          : app.type === "_folder"
            ? `folders/${app.id}`
            : `tasks/${app.id}`;

      await queryApi(endpoint, {
        method: "DELETE",
      });
      router.invalidate();

      const successMessage =
        app.type === "dashboard"
          ? "Dashboard deleted successfully"
          : app.type === "_folder"
            ? "Folder deleted successfully"
            : "Task deleted successfully";

      toast({
        title: "Success",
        description: successMessage,
      });
    } catch (err) {
      if (isRedirect(err)) {
        return navigate(err.options);
      }
      toast({
        title: "Error",
        description: err instanceof Error ? err.message : "Unknown error",
        variant: "error",
      });
    }
  };

  const handleCreateFolder = async () => {
    if (!folderName.trim()) {
      toast({
        title: "Error",
        description: "Folder name is required",
        variant: "error",
      });
      return;
    }

    try {
      await queryApi("folders", {
        method: "POST",
        body: {
          name: folderName.trim(),
          path,
        },
      });
      setFolderDialog(false);
      setFolderName("");
      toast({
        title: "Success",
        description: "Folder created successfully",
      });
      // Refresh the list to show the new folder
      router.invalidate();
    } catch (err) {
      if (isRedirect(err)) {
        return navigate(err.options);
      }

      let errorMessage = "Unknown error";
      if (err instanceof Error) {
        errorMessage = err.message;
        // Check if it's a duplicate folder error
        if (errorMessage.includes("already exists")) {
          errorMessage = `A folder with the name "${folderName.trim()}" already exists in this location. Please choose a different name.`;
        }
      }

      toast({
        title: "Error",
        description: errorMessage,
        variant: "error",
      });
    }
  };

  const handleRenameFolder = async () => {
    if (!renameDialog || !renameName.trim()) {
      toast({
        title: "Error",
        description: "Folder name is required",
        variant: "error",
      });
      return;
    }

    try {
      await queryApi(`folders/${renameDialog.id}/name`, {
        method: "POST",
        body: {
          name: renameName.trim(),
        },
      });
      setRenameDialog(null);
      setRenameName("");
      toast({
        title: "Success",
        description: "Folder renamed successfully",
      });
      // Refresh the list to show the updated folder name
      router.invalidate();
    } catch (err) {
      if (isRedirect(err)) {
        return navigate(err.options);
      }

      let errorMessage = "Unknown error";
      if (err instanceof Error) {
        errorMessage = err.message;
        // Check if it's a duplicate folder error
        if (errorMessage.includes("already exists")) {
          errorMessage = `A folder with the name "${renameName.trim()}" already exists in this location. Please choose a different name.`;
        }
      }

      toast({
        title: "Error",
        description: errorMessage,
        variant: "error",
      });
    }
  };

  const generateBreadcrumbs = () => {
    const pathParts = path.split("/").filter((part) => part !== "");
    const breadcrumbs = [];

    // Add root breadcrumb
    breadcrumbs.push({
      name: "Home",
      path: "/",
      isRoot: true,
    });

    // Add path breadcrumbs
    let currentPath = "";
    for (let i = 0; i < pathParts.length; i++) {
      currentPath += `/${pathParts[i]}`;
      breadcrumbs.push({
        name: pathParts[i],
        path: currentPath + "/",
        isRoot: false,
      });
    }

    return breadcrumbs;
  };

  const handleDragStart = (e: React.DragEvent, item: IApp) => {
    setDraggedItem(item);
    e.dataTransfer.effectAllowed = "move";
    e.dataTransfer.setData("text/plain", ""); // Required for some browsers

    // Create custom drag image showing the whole row
    const draggedRow = e.currentTarget as HTMLElement;
    const originalTable = draggedRow.closest("table") as HTMLTableElement;

    // Create a complete table structure for the drag image
    const dragTable = document.createElement("table");
    const dragTbody = document.createElement("tbody");
    const clonedRow = draggedRow.cloneNode(true) as HTMLElement;

    // Copy table styling and structure
    dragTable.className = originalTable.className;
    dragTable.style.position = "absolute";
    dragTable.style.top = "-9999px";
    dragTable.style.left = "-9999px";
    dragTable.style.width = originalTable.offsetWidth + "px";
    dragTable.style.backgroundColor = "var(--cbgs)";
    dragTable.style.border = "1px solid var(--cbga)";
    dragTable.style.borderRadius = "4px";
    dragTable.style.opacity = "0.9";
    dragTable.style.transform = "rotate(1deg)";
    dragTable.style.boxShadow = "0 4px 12px rgba(0, 0, 0, 0.15)";
    dragTable.style.borderCollapse = "separate";
    dragTable.style.borderSpacing = "0";

    // Add dark mode styles
    if (document.documentElement.classList.contains("dark")) {
      dragTable.style.backgroundColor = "var(--dbgs)";
      dragTable.style.borderColor = "var(--dbga)";
    }

    // Copy column widths from original table
    const originalCells = draggedRow.querySelectorAll("td");
    const clonedCells = clonedRow.querySelectorAll("td");
    originalCells.forEach((originalCell, index) => {
      if (clonedCells[index]) {
        (clonedCells[index] as HTMLElement).style.width =
          originalCell.offsetWidth + "px";
      }
    });

    // Assemble the drag table
    dragTbody.appendChild(clonedRow);
    dragTable.appendChild(dragTbody);

    // Append to body temporarily
    document.body.appendChild(dragTable);

    // Set the custom drag image
    e.dataTransfer.setDragImage(
      dragTable,
      originalTable.offsetWidth / 2,
      draggedRow.offsetHeight / 2,
    );

    // Clean up the clone after a short delay
    setTimeout(() => {
      if (document.body.contains(dragTable)) {
        document.body.removeChild(dragTable);
      }
    }, 0);
  };

  const handleDragEnd = () => {
    setDraggedItem(null);
    setDragOverTarget(null);
  };

  const handleDragOver = (
    e: React.DragEvent,
    targetPath: string,
    targetItem?: IApp,
  ) => {
    e.preventDefault();

    if (!draggedItem) return;

    // Prevent dropping item onto its current location
    if (targetPath === draggedItem.path) {
      e.dataTransfer.dropEffect = "none";
      return;
    }

    // Prevent drag over on invalid targets
    if (targetItem) {
      // Prevent dragging onto itself
      if (draggedItem.id === targetItem.id) {
        e.dataTransfer.dropEffect = "none";
        return;
      }

      // Prevent dragging onto non-folders
      if (targetItem.type !== "_folder") {
        e.dataTransfer.dropEffect = "none";
        return;
      }
    }

    e.dataTransfer.dropEffect = "move";
    setDragOverTarget(targetPath);
  };

  const handleDragLeave = () => {
    setDragOverTarget(null);
  };

  const handleDrop = async (
    e: React.DragEvent,
    targetPath: string,
    targetItem?: IApp,
  ) => {
    e.preventDefault();

    if (!draggedItem) return;

    // Prevent dropping item onto its current location
    if (targetPath === draggedItem.path) {
      toast({
        title: "Error",
        description: "Item is already in this location",
        variant: "error",
      });
      setDragOverTarget(null);
      return;
    }

    // Prevent dropping item onto itself
    if (targetItem && draggedItem.id === targetItem.id) {
      toast({
        title: "Error",
        description: "Cannot drop an item onto itself",
        variant: "error",
      });
      setDragOverTarget(null);
      return;
    }

    // Prevent dropping onto non-folder items (only allow breadcrumb paths and folders)
    if (targetItem && targetItem.type !== "_folder") {
      toast({
        title: "Error",
        description: "Can only drop items into folders",
        variant: "error",
      });
      setDragOverTarget(null);
      return;
    }

    // Prevent dropping folder into itself or its children
    if (draggedItem.type === "_folder") {
      const draggedPath = draggedItem.path;
      if (
        targetPath === draggedPath ||
        targetPath.startsWith(draggedPath + "/")
      ) {
        toast({
          title: "Error",
          description: "Cannot drop a folder into itself or its subfolders",
          variant: "error",
        });
        setDragOverTarget(null);
        return;
      }
    }

    try {
      const requestBody = {
        apps: draggedItem.type !== "_folder" ? [draggedItem.id] : [],
        folders: draggedItem.type === "_folder" ? [draggedItem.id] : [],
        path: targetPath,
      };

      await queryApi("move", {
        method: "POST",
        body: requestBody,
      });

      toast({
        title: "Success",
        description: `"${draggedItem.name}" move to ${targetPath}`,
      });

      // Refresh the list
      router.invalidate();
    } catch (err) {
      if (isRedirect(err)) {
        return navigate(err.options);
      }

      toast({
        title: "Error",
        description: err instanceof Error ? err.message : "Unknown error",
        variant: "error",
      });
    }

    setDragOverTarget(null);
  };

  // Touch support for mobile devices
  const handleTouchStart = (e: React.TouchEvent, item: IApp) => {
    // Prevent default to avoid scrolling
    e.preventDefault();
    setDraggedItem(item);
  };

  const handleTouchEnd = () => {
    setDraggedItem(null);
    setDragOverTarget(null);
  };

  const handleTouchMove = (
    e: React.TouchEvent,
    targetPath: string,
    targetItem?: IApp,
  ) => {
    if (!draggedItem) return;

    // Prevent dropping item onto its current location
    if (targetPath === draggedItem.path) {
      setDragOverTarget(null);
      return;
    }

    // Apply same validation logic as handleDragOver
    if (targetItem) {
      if (draggedItem.id === targetItem.id || targetItem.type !== "_folder") {
        setDragOverTarget(null);
        return;
      }
    }

    const touch = e.touches[0];
    const element = document.elementFromPoint(touch.clientX, touch.clientY);

    if (element && element.closest("[data-drop-target]")) {
      setDragOverTarget(targetPath);
    } else {
      setDragOverTarget(null);
    }
  };

  if (!data) {
    return <div className="p-2">Loading...</div>;
  }

  return (
    <MenuProvider isHome currentPath={path}>
      <Helmet>
        <title>Overview</title>
        <meta
          name="description"
          content="Show a list of all dashboards and tasks"
        />
      </Helmet>

      <div className="pb-4 md:px-4 h-dvh flex flex-col">
        <div className="flex pl-4 pr-2 md:px-0">
          <MenuTrigger className="pr-1.5 py-3 -ml-1.5" />
          <div className="flex-grow flex pb-2 pt-2.5 gap-2 my-2 overflow-x-auto">
            <nav className="flex items-center gap-1 font-semibold font-display">
              {generateBreadcrumbs().map((breadcrumb, index) => (
                <div key={breadcrumb.path} className="flex items-center gap-1">
                  {index > 0 && (
                    <RiArrowRightSLine
                      className="size-4 text-ctext2 dark:text-dtext2"
                      aria-hidden={true}
                    />
                  )}
                  <Link
                    to={"/"}
                    search={
                      breadcrumb.path === "/"
                        ? undefined
                        : { path: breadcrumb.path }
                    }
                    className={cx(
                      "hover:text-cprimary dark:hover:text-dprimary transition-colors duration-200 px-2 py-1 -my-1 -mx-1 rounded",
                      "whitespace-nowrap",
                      {
                        "outline-2 outline-dashed outline-cprimary dark:outline-dprimary":
                          dragOverTarget === breadcrumb.path,
                      },
                    )}
                    onDragOver={(e) => handleDragOver(e, breadcrumb.path)}
                    onDragLeave={handleDragLeave}
                    onDrop={(e) => handleDrop(e, breadcrumb.path)}
                    onTouchMove={(e) => handleTouchMove(e, breadcrumb.path)}
                    data-drop-target
                    data-target-path={breadcrumb.path}
                  >
                    {breadcrumb.name}
                  </Link>
                </div>
              ))}
            </nav>
          </div>
          <div className="flex">
            <Tooltip showArrow={false} content="New Folder">
              <Button
                variant="secondary"
                className="py-2 px-2.5"
                onClick={() => setFolderDialog(true)}
              >
                <RiFolderAddLine

                  className="size-4 shrink-0"
                  aria-hidden={true}
                />
              </Button>
            </Tooltip>
          </div>
        </div>

        <div className="bg-cbgs dark:bg-dbgs rounded-md shadow flex-grow md:p-4 h-[calc(100%-4rem)]">
          {data.apps.length === 0 ? (
            <div className="h-full flex flex-col items-center justify-center">
              <RiLayoutFill
                className="mx-auto size-9 fill-ctext dark:fill-dtext"
                aria-hidden={true}
              />
              <p className="mt-2 mb-3 font-medium text-ctext dark:text-dtext">
                {path === "/"
                  ? "Create a first dashboard"
                  : "Create a dashboard or task"}
              </p>
              <Link to="/new" search={{ path }}>
                <Button>
                  <RiAddFill
                    className="-ml-1 mr-0.5 size-5 shrink-0"
                    aria-hidden={true}
                  />
                  New
                </Button>
              </Link>
            </div>
          ) : (
            <TableRoot className="h-full overflow-y-auto">
              <Table>
                <TableHead>
                  <TableRow>
                    <TableHeaderCell className="text-ctext">
                      Type
                    </TableHeaderCell>
                    <TableHeaderCell
                      onClick={() => handleSort("name" as const)}
                      className="text-ctext dark:text-dtext cursor-pointer hover:underline"
                    >
                      Name <SortIcon field="name" />
                    </TableHeaderCell>
                    <TableHeaderCell
                      className="text-ctext dark:text-dtext hidden md:table-cell cursor-pointer hover:underline"
                      onClick={() => handleSort("created" as const)}
                    >
                      Created <SortIcon field="created" />
                    </TableHeaderCell>
                    <TableHeaderCell
                      className="text-ctext dark:text-dtext hidden md:table-cell cursor-pointer hover:underline"
                      onClick={() => handleSort("updated" as const)}
                    >
                      Updated <SortIcon field="updated" />
                    </TableHeaderCell>
                    <TableHeaderCell className="text-ctext dark:text-dtext  w-16"></TableHeaderCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {data.apps.map((app) => (
                    <TableRow
                      key={app.id}
                      className={cx(
                        "group",
                        "[tbody_&]:odd:bg-cbgs [tbody_&]:odd:dark:bg-dbgs hover:bg-cbga hover:dark:bg-dbga [tbody_&]:odd:hover:bg-cbga [tbody_&]:odd:hover:dark:bg-dbga",
                        "border-b-1 border-solid !border-cbga !dark:border-dbga",
                        {
                          "opacity-50": draggedItem?.id === app.id,
                          "outline-2 outline-dashed outline-cprimary dark:outline-dprimary -outline-offset-2":
                            app.type === "_folder" &&
                            app.path + app.name + "/" === dragOverTarget,
                        },
                      )}
                      draggable
                      onDragStart={(e) => handleDragStart(e, app)}
                      onDragEnd={handleDragEnd}
                      onTouchStart={(e) => handleTouchStart(e, app)}
                      onTouchEnd={handleTouchEnd}
                      {...(app.type === "_folder"
                        ? {
                          onDragOver: (e: React.DragEvent) =>
                            handleDragOver(e, app.path + app.name + "/", app),
                          onDragLeave: handleDragLeave,
                          onDrop: (e: React.DragEvent) =>
                            handleDrop(e, app.path + app.name + "/", app),
                          onTouchMove: (e: React.TouchEvent) =>
                            handleTouchMove(e, app.path + app.name + "/"),
                          "data-drop-target": true,
                          "data-target-path": app.path + app.name + "/",
                        }
                        : {})}
                    >
                      <TableCell className="font-medium text-ctext dark:text-dtext !p-0 group-hover:underline">
                        {app.type === "_folder" ? (
                          <Link
                            to={"/"}
                            search={{ path: app.path + app.name + "/" }}
                            className={cx(
                              "p-4 block w-full text-left rounded transition-colors duration-200",
                              {
                                "bg-blue-100 dark:bg-blue-900":
                                  dragOverTarget === app.path,
                              },
                            )}
                          >
                            <Tooltip
                              showArrow={false}
                              content={
                                <span className="capitalize">{app.type}</span>
                              }
                            >
                              <RiFolderFill
                                className="size-5 fill-ctext2 dark:fill-dtext2 inline -mt-1"
                                aria-hidden={true}
                              />
                            </Tooltip>
                          </Link>
                        ) : (
                          <Link
                            to={
                              app.type === "dashboard"
                                ? "/dashboards/$id"
                                : "/tasks/$id"
                            }
                            params={{ id: app.id }}
                            className="p-4 block"
                          >
                            <Tooltip
                              showArrow={false}
                              content={
                                <span className="capitalize">{app.type}</span>
                              }
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
                        )}
                      </TableCell>
                      <TableCell className="font-medium text-ctext dark:text-dtext !p-0">
                        {app.type === "_folder" ? (
                          <Link
                            to={"/"}
                            search={{ path: app.path + app.name + "/" }}
                            className={cx(
                              "p-4 block w-full text-left rounded transition-colors duration-200",
                              {
                                "bg-blue-100 dark:bg-blue-900":
                                  dragOverTarget === app.path,
                              },
                            )}
                          >
                            <span className="group-hover:underline">
                              {app.name}
                            </span>
                          </Link>
                        ) : (
                          <Link
                            to={
                              app.type === "dashboard"
                                ? "/dashboards/$id"
                                : "/tasks/$id"
                            }
                            params={{ id: app.id }}
                            className="p-4 block"
                          >
                            <span className="group-hover:underline">
                              {app.name}
                            </span>
                            {app.type === "task" ? (
                              app.taskInfo &&
                              (!(app.taskInfo.lastRunSuccess ?? true) ? (
                                <RuntimeTooltip
                                  lastRunAt={app.taskInfo.lastRunAt}
                                  nextRunAt={app.taskInfo.nextRunAt}
                                >
                                  <span className="bg-cerr dark:bg-derr text-ctexti dark:text-dtexti text-xs rounded p-1 ml-2 opacity-60 group-hover:opacity-100 transition-opacity duration-200">
                                    Task Error
                                  </span>
                                </RuntimeTooltip>
                              ) : (
                                app.taskInfo.nextRunAt != null && (
                                  <RuntimeTooltip
                                    lastRunAt={app.taskInfo.lastRunAt}
                                  >
                                    <span className="bg-cprimary dark:bg-dprimary text-ctexti dark:text-dtexti text-xs rounded p-1 ml-2 opacity-60 group-hover:opacity-100 transition-opacity duration-200">
                                      Next Run:{" "}
                                      <RelativeDate
                                        refresh
                                        date={new Date(app.taskInfo.nextRunAt)}
                                      />
                                    </span>
                                  </RuntimeTooltip>
                                )
                              ))
                            ) : app.visibility === "public" ? (
                              <Tooltip
                                showArrow={false}
                                content="This dashboard is public"
                              >
                                <RiGlobalLine className="size-4 inline-block ml-2 -mt-0.5 fill-ctext dark:fill-dtext" />
                              </Tooltip>
                            ) : (
                              app.visibility === "password-protected" && (
                                <Tooltip
                                  showArrow={false}
                                  content={
                                    "This dashboard has a share link protected with a password"
                                  }
                                >
                                  <RiUserSharedLine className="size-4 inline-block ml-2 -mt-0.5 fill-ctext dark:fill-dtext" />
                                </Tooltip>
                              )
                            )}
                          </Link>
                        )}
                      </TableCell>
                      <TableCell className="hidden md:table-cell text-ctext2 dark:text-dtext2 p-0">
                        {app.type === "_folder" ? (
                          <Link
                            to={"/"}
                            search={{ path: app.path + app.name + "/" }}
                            className={cx(
                              "p-4 block w-full text-left rounded transition-colors duration-200",
                              {
                                "bg-blue-100 dark:bg-blue-900":
                                  dragOverTarget === app.path,
                              },
                            )}
                          >
                            <Tooltip
                              showArrow={false}
                              content={new Date(app.createdAt).toLocaleString()}
                            >
                              {new Date(app.createdAt).toLocaleDateString()}
                            </Tooltip>
                          </Link>
                        ) : (
                          <Link
                            to={
                              app.type === "dashboard"
                                ? "/dashboards/$id"
                                : "/tasks/$id"
                            }
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
                        )}
                      </TableCell>
                      <TableCell className="hidden md:table-cell text-ctext2 dark:text-dtext2 p-0">
                        {app.type === "_folder" ? (
                          <Link
                            to={"/"}
                            search={{ path: app.path + app.name + "/" }}
                            className={cx(
                              "p-4 block w-full text-left rounded transition-colors duration-200",
                              {
                                "bg-blue-100 dark:bg-blue-900":
                                  dragOverTarget === app.path,
                              },
                            )}
                          >
                            <Tooltip
                              showArrow={false}
                              content={new Date(app.updatedAt).toLocaleString()}
                            >
                              {new Date(app.updatedAt).toLocaleDateString()}
                            </Tooltip>
                          </Link>
                        ) : (
                          <Link
                            to={
                              app.type === "dashboard"
                                ? "/dashboards/$id"
                                : "/tasks/$id"
                            }
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
                        )}
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex gap-4 justify-end">
                          {app.type === "_folder" ? (
                            <button
                              onClick={() => {
                                setRenameDialog(app);
                                setRenameName(app.name);
                              }}
                            >
                              <Tooltip showArrow={false} content="Rename">
                                <RiPencilLine className="size-5 fill-ctext2 dark:fill-dtext2 inline -mt-1 hover:fill-cprimary dark:hover:fill-dprimary transition-colors duration-200" />
                              </Tooltip>
                            </button>
                          ) : (
                            <Link
                              to={
                                app.type === "dashboard"
                                  ? "/dashboards/$id/edit"
                                  : "/tasks/$id"
                              }
                              params={{ id: app.id }}
                              className={cx(
                                "text-ctext2 dark:text-dtext2 hover:text-ctext dark:hover:text-dtext",
                                "hover:underline transition-colors duration-200",
                              )}
                            >
                              <Tooltip showArrow={false} content="Edit">
                                <RiPencilLine className="size-5 fill-ctext2 dark:fill-dtext2 inline -mt-1 hover:fill-cprimary dark:hover:fill-dprimary transition-colors duration-200" />
                              </Tooltip>
                            </Link>
                          )}
                          <button
                            onClick={() => {
                              setDeleteDialog(app);
                            }}
                          >
                            <Tooltip showArrow={false} content="Delete">
                              <RiDeleteBinLine
                                className="size-5 fill-ctext2 dark:fill-dtext2 inline -mt-1 hover:fill-cerr dark:hover:fill-derr transition-colors duration-200"
                                aria-hidden={true}
                              />
                            </Tooltip>
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

      <Dialog
        open={deleteDialog !== null}
        onOpenChange={(open) => !open && setDeleteDialog(null)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Confirm Deletion</DialogTitle>
            <DialogDescription>
              {deleteDialog &&
                (deleteDialog.type === "dashboard"
                  ? "Are you sure you want to delete the dashboard \"%%\"?"
                  : deleteDialog.type === "_folder"
                    ? "Are you sure you want to delete the folder \"%%\" and all its contents?"
                    : "Are you sure you want to delete the task \"%%\"?"
                ).replace("%%", deleteDialog.name)}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button onClick={() => setDeleteDialog(null)} variant="secondary">
              Cancel
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
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={folderDialog}
        onOpenChange={(open) => !open && setFolderDialog(false)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Create New Folder</DialogTitle>
          </DialogHeader>
          <Input
            placeholder="Folder Name"
            onChange={(e) => setFolderName(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                handleCreateFolder();
              }
            }}
            autoFocus
          />
          <DialogFooter>
            <Button onClick={() => setFolderDialog(false)} variant="secondary">
              Cancel
            </Button>
            <Button onClick={handleCreateFolder}>Create Folder</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={!!renameDialog}
        onOpenChange={(open) => !open && setRenameDialog(null)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Rename Folder</DialogTitle>
          </DialogHeader>
          <Input
            placeholder="Folder Name"
            value={renameName}
            onChange={(e) => setRenameName(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                handleRenameFolder();
              }
            }}
            autoFocus
          />
          <DialogFooter>
            <Button onClick={() => setRenameDialog(null)} variant="secondary">
              Cancel
            </Button>
            <Button onClick={handleRenameFolder}>Rename Folder</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </MenuProvider>
  );
}

function RuntimeTooltip({
  lastRunAt,
  nextRunAt,
  children,
}: {
  lastRunAt?: number | string;
  nextRunAt?: number | string;
  children?: React.ReactNode;
}) {
  if (lastRunAt == null) return children;
  const tooltipContent = (
    <>
      Last Run: <RelativeDate refresh date={new Date(lastRunAt)} />
      {nextRunAt != null && (
        <>
          <br />
          Next Run: <RelativeDate refresh date={new Date(nextRunAt)} />
        </>
      )}
    </>
  );
  return (
    <Tooltip showArrow={false} content={tooltipContent}>
      {children}
    </Tooltip>
  );
}
