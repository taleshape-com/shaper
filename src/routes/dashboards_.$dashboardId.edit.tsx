import { Editor } from "@monaco-editor/react";
import * as monaco from "monaco-editor";

import { z } from "zod";
import { createFileRoute, isRedirect, Link, useNavigate, useRouter } from "@tanstack/react-router";
import { useCallback, useEffect, useRef, useState } from "react";
import { Helmet } from "react-helmet";
import { useAuth } from "../lib/auth";
import { Dashboard } from "../components/dashboard";
import { useDebouncedCallback } from "use-debounce";
import {
  cx,
  focusRing,
  getSearchParamString,
  hasErrorInput,
  varsParamSchema,
} from "../lib/utils";
import { translate } from "../lib/translate";
import { editorStorage } from "../lib/editorStorage";
import { IDashboard, Result } from "../lib/dashboard";
import { Button } from "../components/tremor/Button";
import { useQueryApi } from "../hooks/useQueryApi";
import { MenuProvider } from "../components/MenuProvider";
import { MenuTrigger } from "../components/MenuTrigger";
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

export const Route = createFileRoute("/dashboards_/$dashboardId/edit")({
  validateSearch: z.object({
    vars: varsParamSchema,
  }),
  shouldReload: (match) => {
    return match.cause === "enter";
  },
  loader: async ({
    params: { dashboardId },
    context: {
      queryApi,
    },
  }) => {
    const data = await queryApi(`/api/dashboards/${dashboardId}/query`);
    return data as IDashboard;
  },
  component: DashboardEditor,
});

function DashboardEditor() {
  const params = Route.useParams();
  const { vars } = Route.useSearch();
  const dashboard = Route.useLoaderData();
  const router = useRouter();
  const auth = useAuth();
  const queryApi = useQueryApi();
  const navigate = useNavigate({ from: "/dashboards/$dashboardId/edit" });
  const [editorQuery, setEditorQuery] = useState(dashboard.content);
  const [runningQuery, setRunningQuery] = useState(dashboard.content);
  const [saving, setSaving] = useState(false);
  const [editingName, setEditingName] = useState(false);
  const [name, setName] = useState(dashboard.name);
  const [savingName, setSavingName] = useState(false);
  const [hasVariableError, setHasVariableError] = useState(false);
  const [previewData, setPreviewData] = useState<Result | undefined>(undefined);
  const [previewError, setPreviewError] = useState<string | null>(null);
  const [isPreviewLoading, setIsPreviewLoading] = useState(false);
  const [isDarkMode, setIsDarkMode] = useState(
    window.matchMedia("(prefers-color-scheme: dark)").matches,
  );
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [showRestoreDialog, setShowRestoreDialog] = useState(false);
  const [unsavedContent, setUnsavedContent] = useState<string | null>(null);
  const { toast } = useToast();

  useEffect(() => {
    const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
    const handleChange = (e: MediaQueryListEvent) => {
      setIsDarkMode(e.matches);
    };

    mediaQuery.addEventListener("change", handleChange);
    return () => mediaQuery.removeEventListener("change", handleChange);
  }, []);

  // Check for unsaved changes when component mounts
  useEffect(() => {
    const savedContent = editorStorage.getChanges(params.dashboardId);
    if (savedContent && savedContent !== dashboard.content) {
      setUnsavedContent(savedContent);
      setShowRestoreDialog(true);
    }
  }, [params.dashboardId, dashboard.content]);

  const previewDashboard = useCallback(async () => {
    setPreviewError(null);
    setIsPreviewLoading(true);
    try {
      const searchParams = getSearchParamString(vars);
      const data = await queryApi(`/api/query/dashboard?${searchParams}`, {
        method: "POST",
        body: {
          dashboardId: params.dashboardId,
          content: runningQuery,
        },
      });
      setPreviewData(data);
    } catch (err) {
      if (isRedirect(err)) {
        return navigate(err);
      }
      setPreviewError(err instanceof Error ? err.message : "Unknown error");
    } finally {
      setIsPreviewLoading(false);
    }
  }, [queryApi, params, vars, runningQuery, navigate]);

  const handleRun = useCallback(() => {
    if (isPreviewLoading) {
      return;
    }
    if (editorQuery !== runningQuery) {
      setRunningQuery(editorQuery);
    } else {
      previewDashboard();
    }
  }, [editorQuery, runningQuery, previewDashboard, isPreviewLoading]);

  const handleRunRef = useRef(handleRun);

  useEffect(() => {
    handleRunRef.current = handleRun;
  }, [handleRun]);

  // We handle this command in monac and outside
  // so even if the editor is not focused the shortcut works
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Check for Ctrl+Enter or Cmd+Enter (Mac)
      if ((e.ctrlKey || e.metaKey) && e.key === "Enter") {
        e.preventDefault();
        handleRun();
      }
    };
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [handleRun]);


  // Update textarea onChange handler
  const handleQueryChange = (value: string | undefined) => {
    const newQuery = value || "";
    // Save to localStorage
    editorStorage.saveChanges(params.dashboardId, newQuery);
    setEditorQuery(newQuery);
  };

  const handleSave = useCallback(async () => {
    setSaving(true);
    try {
      await queryApi(
        `/api/dashboards/${params.dashboardId}/query`,
        {
          method: "POST",
          body: { content: editorQuery },
        },
      );
      // Clear localStorage after successful save
      editorStorage.clearChanges(params.dashboardId);
      dashboard.content = editorQuery;
      router.invalidate();
    } catch (err) {
      if (isRedirect(err)) {
        return navigate(err);
      }
      toast({
        title: translate("Error"),
        description:
          err instanceof Error
            ? err.message
            : translate("An error occurred"),
        variant: "error",
      });
    } finally {
      setSaving(false);
    }
  }, [queryApi, params.dashboardId, editorQuery, dashboard, navigate, toast]);

  const handleVarsChanged = useCallback(
    (newVars: any) => {
      navigate({
        search: (old) => ({
          ...old,
          vars: newVars,
        }),
      });
    },
    [navigate],
  );

  const onVariablesEdit = useDebouncedCallback((value) => {
    auth.updateVariables(value).then(
      (ok) => {
        setHasVariableError(!ok);
        if (ok) {
          // Refresh preview when variables change
          previewDashboard();
        }
      },
      () => {
        setHasVariableError(true);
      },
    );
  }, 500);

  const handleSaveName = async (newName: string) => {
    if (newName === dashboard.name) {
      setEditingName(false);
      return;
    }
    setSavingName(true);
    try {
      await queryApi(
        `/api/dashboards/${params.dashboardId}/name`,
        {
          method: "POST",
          body: { name: newName },
        },
      );
      dashboard.name = newName;
      setName(newName);
    } catch (err) {
      if (isRedirect(err)) {
        return navigate(err);
      }
      toast({
        title: translate("Error"),
        description:
          err instanceof Error
            ? err.message
            : translate("An error occurred"),
        variant: "error",
      });
      // Revert name on error
      setName(dashboard.name);
    } finally {
      setSavingName(false);
      setEditingName(false);
    }
  };

  const handleDelete = async () => {
    try {
      await queryApi(`/api/dashboards/${params.dashboardId}`, {
        method: "DELETE",
      });
      // Navigate back to dashboard list
      toast({
        title: translate("Success"),
        description: translate("Dashboard deleted successfully"),
      });
      navigate({ to: "/" });
    } catch (err) {
      if (isRedirect(err)) {
        return navigate(err);
      }
      toast({
        title: translate("Error"),
        description:
          err instanceof Error
            ? err.message
            : translate("An error occurred"),
        variant: "error",
      });
    }
  };

  const handleRestoreUnsavedChanges = () => {
    if (unsavedContent) {
      setEditorQuery(unsavedContent);
      setRunningQuery(unsavedContent);
    }
    setShowRestoreDialog(false);
  };

  const handleDiscardUnsavedChanges = () => {
    editorStorage.clearChanges(params.dashboardId);
    setShowRestoreDialog(false);
  };

  // Load initial preview
  useEffect(() => {
    previewDashboard();
  }, [previewDashboard]);

  return (
    <MenuProvider>
      <Helmet>
        <title>{translate("Edit Dashboard")} - {dashboard.name}</title>
      </Helmet>

      <div className="h-dvh flex flex-col">
        <div className="h-[42dvh] flex flex-col overflow-y-hidden max-h-[90dvh] min-h-[12dvh] resize-y shrink-0 shadow-sm dark:shadow-none">
          <div className="flex items-center p-2 border-b border-cb dark:border-none">
            <MenuTrigger className="pr-2">
              <div className="mt-6 px-4">
                <label className="block">
                  <p className="text-lg font-medium font-display ml-1 mb-2">
                    {translate("Variables")}
                  </p>
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
                <Button
                  onClick={() => setShowDeleteDialog(true)}
                  variant="destructive"
                  className="mt-4"
                >
                  {translate("Delete Dashboard")}
                </Button>
              </div>
            </MenuTrigger>

            {editingName ? (
              <form
                onSubmit={(e) => {
                  e.preventDefault();
                  const input = e.currentTarget.querySelector("input");
                  if (input) {
                    input.blur();
                  }
                }}
                className="inline-block flex-grow"
              >
                <input
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  onBlur={() => {
                    if (!savingName) {
                      handleSaveName(name);
                    }
                  }}
                  className={cx(
                    "text-xl font-semibold font-display px-1 py-0 border rounded",
                    "bg-cbgs dark:bg-dbgs border-cb dark:border-db shadow-sm outline-none ring-0 rounded-md",
                    focusRing,
                  )}
                  autoFocus
                  disabled={savingName}
                />
              </form>
            ) : (
              <div className="hidden sm:block flex-grow">
                <Tooltip
                  showArrow={false}
                  asChild
                  content={translate("Click to edit dashboard name")}
                >
                  <h1
                    className="text-xl font-semibold font-display cursor-pointer hover:bg-cbga dark:hover:bg-dbga px-1 rounded inline-block"
                    onClick={() => setEditingName(true)}
                  >
                    {name}
                  </h1>
                </Tooltip>
              </div>
            )}

            <Link
              to="/dashboards/$dashboardId"
              params={{ dashboardId: params.dashboardId }}
              search={() => ({ vars })}
              className="text-sm text-ctext2 dark:text-dtext2 hover:text-ctext dark:hover:text-dtext hover:underline transition-colors duration-200 flex-grow sm:grow-0"
            >
              {translate("View Dashboard")}
            </Link>

            <div className="space-x-2">
              <Tooltip
                showArrow={false}
                asChild
                content="Save Dashboard"
              >
                <Button
                  onClick={handleSave}
                  className={cx("ml-2", { "hidden": editorQuery === dashboard.content })}
                  disabled={saving || editorQuery === dashboard.content}
                  isLoading={saving}
                  variant='secondary'
                >
                  {translate("Save")}
                </Button>
              </Tooltip>
              <Tooltip
                showArrow={false}
                asChild
                content="Press Ctrl + Enter to run"
              >
                <Button
                  onClick={handleRun}
                  disabled={isPreviewLoading}
                  isLoading={isPreviewLoading}
                >
                  {translate("Run")}
                </Button>
              </Tooltip>
            </div>
          </div>

          <div className="flex-grow">
            <Editor
              height="100%"
              defaultLanguage="sql"
              value={editorQuery}
              onChange={handleQueryChange}
              theme={isDarkMode ? "vs-dark" : "light"}
              options={{
                minimap: { enabled: false },
                fontSize: 14,
                lineNumbers: "on",
                scrollBeyondLastLine: true,
                wordWrap: "on",
                automaticLayout: true,
                formatOnPaste: true,
                formatOnType: true,
                suggestOnTriggerCharacters: true,
                quickSuggestions: true,
                tabSize: 2,
                bracketPairColorization: { enabled: true },
              }}
              onMount={(editor) => {
                editor.addCommand(
                  monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter,
                  () => {
                    handleRunRef.current();
                  },
                );
              }}
            />
          </div>
        </div>

        <div className="flex-grow overflow-y-auto relative">
          {previewError && (
            <div className="fixed w-full h-full p-4 z-50 backdrop-blur-sm flex justify-center">
              <div className="p-4 bg-red-100 text-red-700 rounded mt-32 h-fit">
                {previewError}
              </div>
            </div>
          )}
          <Dashboard
            vars={vars}
            hash={auth.hash}
            getJwt={auth.getJwt}
            onVarsChanged={handleVarsChanged}
            data={previewData} // Pass preview data directly to Dashboard
            loading={isPreviewLoading}
          />
        </div>
      </div>

      <Dialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{translate("Confirm Deletion")}</DialogTitle>
            <DialogDescription>
              {translate('Are you sure you want to delete the dashboard "%%"?').replace(
                "%%",
                dashboard.name,
              )}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button onClick={() => setShowDeleteDialog(false)}>
              {translate("Cancel")}
            </Button>
            <Button
              variant="destructive"
              onClick={() => {
                handleDelete();
                setShowDeleteDialog(false);
              }}
            >
              {translate("Delete")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={showRestoreDialog} onOpenChange={setShowRestoreDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{translate("Unsaved Changes")}</DialogTitle>
            <DialogDescription>
              {translate("There are unsaved previous edits. Do you want to restore them?")}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button 
              variant="secondary"
              onClick={handleDiscardUnsavedChanges}
            >
              {translate("Discard")}
            </Button>
            <Button
              onClick={handleRestoreUnsavedChanges}
            >
              {translate("Restore")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </MenuProvider>
  );
}
