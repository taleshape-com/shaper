// SPDX-License-Identifier: MPL-2.0

import { z } from "zod";
import { createFileRoute, isRedirect, Link, useNavigate, useRouter } from "@tanstack/react-router";
import { useCallback, useEffect, useState } from "react";
import { Helmet } from "react-helmet";
import { RiPencilLine, RiCloseLine, RiArrowDownSLine } from "@remixicon/react";
import { useAuth, getJwt } from "../lib/auth";
import { Dashboard } from "../components/dashboard";
import {
  cx,
  focusRing,
  getSearchParamString,
  varsParamSchema,
} from "../lib/utils";
import { translate } from "../lib/translate";
import { editorStorage } from "../lib/editorStorage";
import { IDashboard, Result } from "../lib/types";
import { Button } from "../components/tremor/Button";
import { useQueryApi } from "../hooks/useQueryApi";
import { MenuProvider } from "../components/providers/MenuProvider";
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
import { VariablesMenu } from "../components/VariablesMenu";
import { PublicLink } from "../components/PublicLink";
import { SqlEditor } from "../components/SqlEditor";
import { PreviewError } from "../components/PreviewError";
import "../lib/editorInit";

export const Route = createFileRoute("/dashboards_/$id/edit")({
  validateSearch: z.object({
    vars: varsParamSchema,
  }),
  shouldReload: (match) => {
    return match.cause === "enter";
  },
  loader: async ({
    params: { id },
    context: {
      queryApi,
    },
  }) => {
    const data = await queryApi(`dashboards/${id}/query`);
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
  const navigate = useNavigate({ from: "/dashboards/$id/edit" });
  const [editorQuery, setEditorQuery] = useState(dashboard.content);
  const [runningQuery, setRunningQuery] = useState(dashboard.content);
  const [saving, setSaving] = useState(false);
  const [editingName, setEditingName] = useState(false);
  const [name, setName] = useState(dashboard.name);
  const [savingName, setSavingName] = useState(false);
  const [previewData, setPreviewData] = useState<Result | undefined>(undefined);
  const [previewError, setPreviewError] = useState<string | null>(null);
  const [isPreviewLoading, setIsPreviewLoading] = useState(false);
  const [loadDuration, setLoadDuration] = useState<number | null>(null);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [showVisibilityDialog, setShowVisibilityDialog] = useState(false);
  const [showRestoreDialog, setShowRestoreDialog] = useState(false);
  const [unsavedContent, setUnsavedContent] = useState<string | null>(null);
  const { toast } = useToast();

  // Check for unsaved changes when component mounts
  useEffect(() => {
    const savedContent = editorStorage.getChanges(params.id);
    if (savedContent && savedContent !== dashboard.content) {
      setUnsavedContent(savedContent);
      setShowRestoreDialog(true);
    }
  }, [params.id, dashboard.content]);

  const previewDashboard = useCallback(async () => {
    setPreviewError(null);
    setIsPreviewLoading(true);
    setLoadDuration(null); // Reset previous duration
    const startTime = Date.now();
    try {
      const searchParams = getSearchParamString(vars);
      const data = await queryApi(`run/dashboard?${searchParams}`, {
        method: "POST",
        body: {
          dashboardId: params.id,
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

  const handleQueryChange = (value: string | undefined) => {
    const newQuery = value || "";
    // Save to localStorage
    editorStorage.saveChanges(params.id, newQuery);
    setEditorQuery(newQuery);
  };

  const handleSave = useCallback(async () => {
    setSaving(true);
    try {
      await queryApi(
        `dashboards/${params.id}/query`,
        {
          method: "POST",
          body: { content: editorQuery },
        },
      );
      // Clear localStorage after successful save
      editorStorage.clearChanges(params.id);
      dashboard.content = editorQuery;
      router.invalidate();
    } catch (err) {
      if (isRedirect(err)) {
        return navigate(err.options);
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
  }, [queryApi, params.id, editorQuery, dashboard, navigate, toast, router]);

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


  const handleSaveName = async (newName: string) => {
    if (newName === dashboard.name) {
      setEditingName(false);
      return;
    }
    setSavingName(true);
    try {
      await queryApi(
        `dashboards/${params.id}/name`,
        {
          method: "POST",
          body: { name: newName },
        },
      );
      dashboard.name = newName;
      setName(newName);
    } catch (err) {
      if (isRedirect(err)) {
        return navigate(err.options);
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
      await queryApi(`dashboards/${params.id}`, {
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
        return navigate(err.options);
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

  const handleVisibilityChange = async () => {
    try {
      await queryApi(
        `dashboards/${params.id}/visibility`,
        {
          method: "POST",
          body: { visibility: dashboard.visibility === 'public' ? 'private' : 'public' },
        },
      );
      toast({
        title: translate(dashboard.visibility === 'public' ? "Dashboard unshared" : "Dashboard made public"),
        description: translate(dashboard.visibility === 'public' ? "The dashboard is not publicly accessible anymore." : "Try the link in the sidebar"),
      });
      dashboard.visibility = dashboard.visibility === 'public' ? 'private' : 'public';
      router.invalidate();
    } catch (err) {
      if (isRedirect(err)) {
        return navigate(err.options);
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
    editorStorage.clearChanges(params.id);
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
              <VariablesMenu
                onVariablesChange={previewDashboard}
              />
              {dashboard.visibility && (
                <div className="my-2 px-4">
                  <Button
                    onClick={() => setShowVisibilityDialog(true)}
                    variant="secondary"
                    className="mt-4 capitalize"
                  >
                    {translate(dashboard.visibility)}
                    <RiArrowDownSLine className="size-4 inline ml-1.5 mt-0.5 fill-ctext2 dark:fill-dtext2" />
                  </Button>
                  {dashboard.visibility === 'public' && (
                    <PublicLink href={`../../view/${params.id}`} />
                  )}
                </div>
              )}
              <Button
                onClick={() => setShowDeleteDialog(true)}
                variant="destructive"
                className="mt-4 mx-4"
              >
                {translate("Delete Dashboard")}
              </Button>
              {loadDuration && (
                <div className="text-xs text-ctext2 dark:text-dtext2 mt-4 mx-4 opacity-85">
                  <span>
                    Load time: {loadDuration >= 1000 ? `${(loadDuration / 1000).toFixed(2)}s` : `${loadDuration}ms`}
                  </span>
                </div>
              )}
            </MenuTrigger>

            {editingName ? (
              <form
                onSubmit={(e) => {
                  e.preventDefault();
                  const input = e.currentTarget.querySelector("input");
                  if (input && !savingName) {
                    handleSaveName(name);
                  }
                }}
                className="flex flex-grow"
              >
                <input
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className={cx(
                    "text-lg font-semibold font-display px-2 py-0.5 border rounded",
                    "bg-cbgs dark:bg-dbgs border-cb dark:border-db shadow-sm outline-none ring-0 rounded-md w-96 max-w-[calc(100%-2rem)]",
                    focusRing,
                  )}
                  autoFocus
                  disabled={savingName}
                />
                <Button
                  variant="destructive"
                  type="reset"
                  className="ml-2"
                  onClick={(e) => {
                    e.preventDefault();
                    setEditingName(false);
                    setName(dashboard.name);
                  }}
                >
                  <RiCloseLine className="size-5" />
                </Button>
                <Button
                  type="submit"
                  variant="primary"
                  className="inline ml-2"
                  disabled={savingName}
                  isLoading={savingName}
                >
                  {translate("Save")}
                </Button>
              </form>
            ) : (
              <div className="hidden sm:block flex-grow">
                <Tooltip
                  showArrow={false}
                  asChild
                  content={translate("Click to edit dashboard name")}
                >
                  <h1
                    className="text-xl font-semibold font-display cursor-pointer hover:bg-cbga dark:hover:bg-dbga px-2 py-0.5 rounded inline-block"
                    onClick={() => setEditingName(true)}
                  >
                    {name}
                    <RiPencilLine className="size-4 inline ml-1.5 mb-1" />
                  </h1>
                </Tooltip>
              </div>
            )}

            <Link
              to="/dashboards/$id"
              params={{ id: params.id }}
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
            <SqlEditor
              onChange={handleQueryChange}
              onRun={handleRun}
              content={editorQuery}
            />
          </div>
        </div>

        <div className="flex-grow overflow-y-auto relative">
          {previewError && (
            <PreviewError>{previewError}</PreviewError>
          )}
          <Dashboard
            vars={vars}
            hash={auth.hash}
            getJwt={getJwt}
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

      <Dialog open={showVisibilityDialog} onOpenChange={setShowVisibilityDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{translate(dashboard.visibility === 'public' ? "Do you want to unshare the dashboard?" : "Do you want to share the dashboard publicly?")}</DialogTitle>
            <DialogDescription>
              {translate(dashboard.visibility === 'public' ? "Are you sure you want to remove public access to the dasboard?" : "Are you sure you want to make the dashboard visible to everyone?")}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              onClick={() => setShowVisibilityDialog(false)}
              variant="secondary"
            >
              {translate("Cancel")}
            </Button>
            <Button
              variant="primary"
              onClick={() => {
                handleVisibilityChange();
                setShowVisibilityDialog(false);
              }}
            >
              {translate(dashboard.visibility === "public" ? "Unshare" : "Make Public")}
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
