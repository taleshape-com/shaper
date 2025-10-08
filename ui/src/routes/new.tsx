// SPDX-License-Identifier: MPL-2.0

import { z } from "zod";
import {
  createFileRoute,
  isRedirect,
  useNavigate,
  Link,
} from "@tanstack/react-router";
import { useCallback, useEffect, useState } from "react";
import { Helmet } from "react-helmet";
import { useAuth, getJwt } from "../lib/auth";
import { Dashboard } from "../components/dashboard";
import { getSearchParamString, isMac, varsParamSchema, cx } from "../lib/utils";
import { editorStorage } from "../lib/editorStorage";
import { Button } from "../components/tremor/Button";
import { useQueryApi } from "../hooks/useQueryApi";
import { MenuProvider } from "../components/providers/MenuProvider";
import { MenuTrigger } from "../components/MenuTrigger";
import { Result } from "../lib/types";
import { useToast } from "../hooks/useToast";
import { Tooltip } from "../components/tremor/Tooltip";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../components/tremor/Dialog";
import { RiCodeSSlashFill, RiBarChart2Line, RiArrowRightSLine } from "@remixicon/react";
import { Input } from "../components/tremor/Input";
import { VariablesMenu } from "../components/VariablesMenu";
import { SqlEditor } from "../components/SqlEditor";
import { PreviewError } from "../components/PreviewError";
import { TaskResults, TaskResult } from "../components/TaskResults";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../components/tremor/Select";
import "../lib/editorInit";
import { getSystemConfig } from "../lib/system";

const defaultDashboardQuery = `SELECT 'Dashboard Title'::SECTION;

SELECT 'Label'::LABEL;
SELECT 'Hello World';

SELECT
  col0::XAXIS,
  col1::BARCHART,
FROM (
  VALUES
  (1, 10),
  (2, 20),
  (3, 30),
);`;

const defaultTaskQuery = `-- Tasks must start with a SCHEDULE statement that defines when the task runs.
-- Examples:

-- Every hour:
-- SELECT (INTERVAL '1h')::SCHEDULE;

-- Every day at 1am:
-- SELECT (date_trunc('day', now()) + INTERVAL '25h')::SCHEDULE;

-- Every Monday at 1am:
-- SELECT (date_trunc('week', now()) + INTERVAL '7days 1hour')::SCHEDULE;

-- Never run automatically:
SELECT NULL::SCHEDULE;
`;

// LocalStorage key for storing the app type preference
const APP_TYPE_STORAGE_KEY = "shaper-new-app-type";

// Utility functions for localStorage
const getStoredAppType = (): "dashboard" | "task" => {
  try {
    const stored = localStorage.getItem(APP_TYPE_STORAGE_KEY);
    return stored === "task" ? "task" : "dashboard";
  } catch {
    return "dashboard";
  }
};

const setStoredAppType = (type: "dashboard" | "task") => {
  try {
    localStorage.setItem(APP_TYPE_STORAGE_KEY, type);
  } catch {
    // Ignore localStorage errors
  }
};

const clearStoredAppType = () => {
  try {
    localStorage.removeItem(APP_TYPE_STORAGE_KEY);
  } catch {
    // Ignore localStorage errors
  }
};

export const Route = createFileRoute("/new")({
  validateSearch: z.object({
    vars: varsParamSchema,
    path: z.string().optional(),
  }),
  component: NewDashboard,
});

function NewDashboard () {
  const { vars, path } = Route.useSearch();
  const auth = useAuth();
  const queryApi = useQueryApi();
  const navigate = useNavigate({ from: "/new" });
  const [appType, setAppType] = useState<"dashboard" | "task">(() =>
    getStoredAppType(),
  );
  const [editorQuery, setEditorQuery] = useState("");
  const [runningQuery, setRunningQuery] = useState("");
  const [creating, setCreating] = useState(false);
  const [previewData, setPreviewData] = useState<Result | undefined>(undefined);
  const [taskData, setTaskData] = useState<TaskResult | undefined>(undefined);
  const [previewError, setPreviewError] = useState<string | null>(null);
  const [isPreviewLoading, setIsPreviewLoading] = useState(false);
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [showDiscardDialog, setShowDiscardDialog] = useState(false);
  const [dashboardName, setDashboardName] = useState("");
  const [loadDuration, setLoadDuration] = useState<number | null>(null);
  const { toast } = useToast();

  // Check for unsaved changes when component mounts or type changes
  useEffect(() => {
    const unsavedContent = editorStorage.getChanges("new");
    if (unsavedContent) {
      setEditorQuery(unsavedContent);
      setRunningQuery(unsavedContent);
    } else {
      // Set default content based on app type
      const defaultContent =
        appType === "task" ? defaultTaskQuery : defaultDashboardQuery;
      setEditorQuery(defaultContent);
      setRunningQuery(defaultContent);
    }
  }, [appType]);

  const previewDashboard = useCallback(async () => {
    if (!runningQuery.trim()) {
      return;
    }
    setPreviewError(null);
    setIsPreviewLoading(true);
    setLoadDuration(null); // Reset previous duration
    const startTime = Date.now();
    try {
      const searchParams = getSearchParamString(vars);
      const data = await queryApi(`run/dashboard?${searchParams}`, {
        method: "POST",
        body: {
          content: runningQuery,
        },
      });
      setPreviewData(data);
    } catch (err) {
      if (isRedirect(err)) {
        return navigate(err.options);
      }
      setPreviewError(err instanceof Error ? err.message : "Unknown error");
    } finally {
      const duration = startTime ? Date.now() - startTime : null;
      setLoadDuration(duration);
      setIsPreviewLoading(false);
    }
  }, [queryApi, vars, runningQuery, navigate]);

  const runTask = useCallback(async () => {
    setPreviewError(null);
    setIsPreviewLoading(true);
    try {
      const data = await queryApi("run/task", {
        method: "POST",
        body: {
          content: editorQuery,
        },
      });
      setTaskData(data);
    } catch (err) {
      if (isRedirect(err)) {
        return navigate(err.options);
      }
      setPreviewError(err instanceof Error ? err.message : "Unknown error");
    } finally {
      setIsPreviewLoading(false);
    }
  }, [queryApi, editorQuery, navigate]);

  useEffect(() => {
    if (appType === "dashboard") {
      previewDashboard();
    }
  }, [previewDashboard, appType]);

  const handleRun = useCallback(() => {
    if (appType === "task") {
      runTask();
    } else {
      if (isPreviewLoading) {
        return;
      }
      if (editorQuery !== runningQuery) {
        setRunningQuery(editorQuery);
      } else {
        previewDashboard();
      }
    }
  }, [
    editorQuery,
    runningQuery,
    previewDashboard,
    runTask,
    isPreviewLoading,
    appType,
  ]);

  const handleTypeChange = useCallback(
    (newType: string) => {
      const type = newType as "dashboard" | "task";
      setAppType(type);
      setStoredAppType(type);

      // Clear results when switching types
      setPreviewData(undefined);
      setTaskData(undefined);
      setPreviewError(null);

      // Auto-run dashboard when switching to it
      if (type === "dashboard") {
        setTimeout(() => previewDashboard(), 0);
      }
    },
    [previewDashboard],
  );

  const handleQueryChange = (value: string | undefined) => {
    const newQuery = value || "";
    const currentDefaultQuery =
      appType === "task" ? defaultTaskQuery : defaultDashboardQuery;

    // Save to localStorage
    if (newQuery !== currentDefaultQuery && newQuery.trim() !== "") {
      editorStorage.saveChanges("new", newQuery);
    } else {
      editorStorage.clearChanges("new");
    }
    setEditorQuery(newQuery);
  };

  const handleCreate = useCallback(async () => {
    if (!dashboardName.trim()) {
      return;
    }

    setCreating(true);
    try {
      if (appType === "task") {
        const { id } = await queryApi("tasks", {
          method: "POST",
          body: {
            name: dashboardName,
            content: editorQuery,
            path: path || "/",
          },
        });
        // Clear localStorage after successful save
        editorStorage.clearChanges("new");
        clearStoredAppType(); // Reset the app type preference

        // Navigate to the task edit page
        navigate({
          replace: true,
          to: "/tasks/$id",
          params: { id },
        });
      } else {
        const { id } = await queryApi("dashboards", {
          method: "POST",
          body: {
            name: dashboardName,
            content: editorQuery,
            path: path || "/",
          },
        });
        // Clear localStorage after successful save
        editorStorage.clearChanges("new");
        clearStoredAppType(); // Reset the app type preference

        // Navigate to the dashboard edit page
        navigate({
          replace: true,
          to: "/dashboards/$id/edit",
          params: { id },
          search: () => ({ vars }),
        });
      }
    } catch (err) {
      if (isRedirect(err)) {
        return navigate(err.options);
      }
      toast({
        title: "Error",
        description: err instanceof Error ? err.message : "An error occurred",
        variant: "error",
      });
      setCreating(false);
      setShowCreateDialog(false);
    }
  }, [queryApi, editorQuery, navigate, vars, toast, dashboardName, appType, path]);

  const handleVarsChanged = useCallback(
    (newVars: any) => {
      navigate({
        search: (old: any) => ({
          ...old,
          vars: newVars,
        }),
      });
    },
    [navigate],
  );

  const handleDiscardChanges = useCallback(() => {
    const defaultContent =
      appType === "task" ? defaultTaskQuery : defaultDashboardQuery;
    editorStorage.clearChanges("new");
    setEditorQuery(defaultContent);
    setRunningQuery(defaultContent);
    setShowDiscardDialog(false);
    // Clear results when discarding
    setPreviewData(undefined);
    setTaskData(undefined);
    setPreviewError(null);
  }, [appType]);

  // Helper to check if content has been modified
  const hasUnsavedChanges = () => {
    const defaultContent =
      appType === "task" ? defaultTaskQuery : defaultDashboardQuery;
    return editorQuery !== defaultContent;
  };

  const generateBreadcrumbs = () => {
    const pathParts = (path || "/").split("/").filter((part) => part !== "");
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

  return (
    <MenuProvider isNewPage currentPath={path || "/"}>
      <Helmet>
        <title>{appType === "task" ? "New Task" : "New Dashboard"}</title>
      </Helmet>

      <div className="h-dvh flex flex-col">
        <div className="h-[42dvh] flex flex-col overflow-y-hidden max-h-[90dvh] min-h-[12dvh] resize-y shrink-0 shadow-sm dark:shadow-none">
          <div className="flex items-center px-2 border-b border-cb dark:border-none">
            <MenuTrigger className="pr-2">
              {appType === "dashboard" && (
                <VariablesMenu onVariablesChange={previewDashboard} />
              )}
              {appType === "dashboard" && loadDuration && (
                <div className="text-xs text-ctext2 dark:text-dtext2 mt-4 mx-4 opacity-85">
                  <span>
                    Load time:{" "}
                    {loadDuration >= 1000
                      ? `${(loadDuration / 1000).toFixed(2)}s`
                      : `${loadDuration}ms`}
                  </span>
                </div>
              )}
            </MenuTrigger>

            <div className="flex-grow flex pb-2 pt-2 gap-2 overflow-x-auto">
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
                      className="hover:text-cprimary dark:hover:text-dprimary transition-colors duration-200 px-2 py-1 -my-1 -mx-1 rounded whitespace-nowrap"
                    >
                      {breadcrumb.name}
                    </Link>
                  </div>
                ))}
              </nav>
              <h1 className="flex items-center gap-2 font-display">
                <RiArrowRightSLine
                  className="size-4 text-ctext2 dark:text-dtext2 mr-0.5"
                  aria-hidden={true}
                />
                New
                {getSystemConfig().tasksEnabled ? (
                  <Select value={appType} onValueChange={handleTypeChange}>
                    <SelectTrigger className="w-36">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="dashboard">
                        <RiBarChart2Line
                          className="size-5 fill-ctext2 dark:fill-dtext2 inline -mt-1 mr-1.5"
                          aria-hidden={true}
                        />
                        Dashboard
                      </SelectItem>
                      <SelectItem value="task">
                        <RiCodeSSlashFill
                          className="size-5 fill-ctext2 dark:fill-dtext2 inline -mt-1 mr-2"
                          aria-hidden={true}
                        />
                        Task
                      </SelectItem>
                    </SelectContent>
                  </Select>
                ) : (
                  " Dashboard"
                )}
              </h1>

            </div>

            <Tooltip
              showArrow={false}
              asChild
              content={`Create ${appType === "task" ? "Task" : "Dashboard"}`}
            >
              <Button
                onClick={() => setShowCreateDialog(true)}
                disabled={creating}
                variant="secondary"
                className="ml-4"
              >
                Create
              </Button>
            </Tooltip>
            <Tooltip showArrow={false} asChild content="Discard Changes">
              <Button
                onClick={() => setShowDiscardDialog(true)}
                className={cx("ml-2", { hidden: !hasUnsavedChanges() })}
                disabled={!hasUnsavedChanges()}
                variant="destructive"
              >
                Discard
              </Button>
            </Tooltip>
            <Tooltip
              showArrow={false}
              asChild
              content={`Press ${isMac() ? "⌘" : "Ctrl"} + Enter to run`}
            >
              <Button
                onClick={handleRun}
                disabled={isPreviewLoading}
                isLoading={isPreviewLoading}
                className="ml-2"
              >
                Run
              </Button>
            </Tooltip>
          </div>

          <div className="flex-grow">
            <SqlEditor
              onChange={handleQueryChange}
              onRun={handleRun}
              content={editorQuery}
            />
          </div>
        </div>

        <div className="flex-grow overflow-y-auto relative">
          {previewError && <PreviewError>{previewError}</PreviewError>}
          {appType === "dashboard" ? (
            <Dashboard
              vars={vars}
              hash={auth.hash}
              getJwt={getJwt}
              onVarsChanged={handleVarsChanged}
              data={previewData}
              loading={isPreviewLoading}
            />
          ) : (
            <TaskResults data={taskData} loading={isPreviewLoading} />
          )}
        </div>
      </div>

      <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {`Create ${appType === "task" ? "Task" : "Dashboard"}`}
            </DialogTitle>
          </DialogHeader>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              handleCreate();
            }}
            className="space-y-4 mt-4"
          >
            <div>
              <Input
                id="dashboardName"
                value={dashboardName}
                onChange={(e) => setDashboardName(e.target.value)}
                placeholder={`Enter a name for the ${appType === "task" ? "task" : "dashboard"}`}
                autoFocus
                required
              />
            </div>
            <DialogFooter>
              <Button
                type="button"
                onClick={() => setShowCreateDialog(false)}
                variant="secondary"
              >
                Cancel
              </Button>
              <Button
                type="submit"
                disabled={creating || !dashboardName.trim()}
                isLoading={creating}
              >
                Create
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog open={showDiscardDialog} onOpenChange={setShowDiscardDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Discard Changes</DialogTitle>
            <DialogDescription>
              Are you sure you want to discard your changes and reset to the
              default content? This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button onClick={() => setShowDiscardDialog(false)}>Cancel</Button>
            <Button variant="destructive" onClick={handleDiscardChanges}>
              Discard
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </MenuProvider>
  );
}
