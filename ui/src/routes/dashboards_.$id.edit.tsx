// SPDX-License-Identifier: MPL-2.0

import { z } from "zod";
import {
  createFileRoute,
  isRedirect,
  Link,
  useNavigate,
  useRouter,
} from "@tanstack/react-router";
import { useCallback, useEffect, useState, useRef } from "react";
import { Helmet } from "react-helmet";
import {
  RiPencilLine,
  RiCloseLine,
  RiArrowDownSLine,
  RiEyeLine,
  RiEyeOffLine,
  RiFileCopyLine,
  RiRefreshLine,
  RiExternalLinkLine,
} from "@remixicon/react";
import { useAuth, getJwt } from "../lib/auth";
import { Dashboard } from "../components/dashboard";
import {
  cx,
  focusRing,
  getSearchParamString,
  varsParamSchema,
  copyToClipboard,
} from "../lib/utils";
import { editorStorage } from "../lib/editorStorage";
import { IDashboard, Result } from "../lib/types";
import { Button } from "../components/tremor/Button";
import { useQueryApi } from "../hooks/useQueryApi";
import { MenuProvider } from "../components/providers/MenuProvider";
import { MenuTrigger } from "../components/MenuTrigger";
import { useToast } from "../hooks/useToast";
import { Tooltip } from "../components/tremor/Tooltip";
import { generatePassword } from "../lib/passwordUtils";
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
import { getSystemConfig } from "../lib/system";

const MIN_SHOW_LOADING = 300;

export const Route = createFileRoute("/dashboards_/$id/edit")({
  validateSearch: z.object({
    vars: varsParamSchema,
  }),
  shouldReload: (match) => {
    return match.cause === "enter";
  },
  loader: async ({ params: { id }, context: { queryApi } }) => {
    const data = await queryApi(`dashboards/${id}/info`);
    return data as IDashboard;
  },
  component: DashboardEditor,
});

function DashboardEditor () {
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
  const [showPasswordSuccessDialog, setShowPasswordSuccessDialog] =
		useState(false);
  const [showSuccessPassword, setShowSuccessPassword] = useState(false);
  const [showDiscardDialog, setShowDiscardDialog] = useState(false);
  const [selectedVisibility, setSelectedVisibility] = useState<string>(
    dashboard.visibility || "private",
  );
  const [password, setPassword] = useState<string>("");
  const [showPassword, setShowPassword] = useState(false);
  const [savingPassword, setSavingPassword] = useState(false);
  const [generatingPassword, setGeneratingPassword] = useState(false);
  const { toast } = useToast();
  const systemConfig = getSystemConfig();

  // Track the current AbortController for preview requests
  const previewAbortRef = useRef<AbortController | null>(null);

  // Ref for dashboard ID text selection
  const dashboardIdRef = useRef<HTMLElement>(null);

  // Check for unsaved changes when component mounts and restore directly
  useEffect(() => {
    const savedContent = editorStorage.getChanges(params.id);
    if (savedContent && savedContent !== dashboard.content) {
      setEditorQuery(savedContent);
      setRunningQuery(savedContent);
    }
  }, [params.id, dashboard.content]);

  const previewDashboard = useCallback(async () => {
    // Abort any in-flight preview request
    if (previewAbortRef.current) {
      previewAbortRef.current.abort();
    }
    const abortController = new AbortController();
    previewAbortRef.current = abortController;

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
        signal: abortController.signal,
      });
      const duration = Date.now() - startTime;
      await new Promise<void>((resolve) => {
        setTimeout(
          () => {
            resolve();
          },
          Math.max(0, MIN_SHOW_LOADING - duration),
        );
      });

      // Only apply result if this is the latest request
      if (previewAbortRef.current === abortController) {
        setLoadDuration(duration);
        setPreviewData(data);
      }
    } catch (err: unknown) {
      if ((err as any)?.name === "AbortError") {
        return; // ignore aborts
      }
      if (isRedirect(err)) {
        return navigate(err.options);
      }
      setPreviewError(err instanceof Error ? err.message : "Unknown error");
      setLoadDuration(Date.now() - startTime);
    } finally {
      if (previewAbortRef.current === abortController) {
        setIsPreviewLoading(false);
      }
    }
  }, [queryApi, params, vars, runningQuery, navigate]);

  const handleRun = useCallback(() => {
    if (editorQuery !== runningQuery) {
      setRunningQuery(editorQuery);
    } else {
      previewDashboard();
    }
  }, [editorQuery, runningQuery, previewDashboard]);

  const handleQueryChange = (value: string | undefined) => {
    const newQuery = value || "";
    // Save to localStorage
    editorStorage.saveChanges(params.id, newQuery);
    setEditorQuery(newQuery);
  };

  const handleSave = useCallback(async () => {
    setSaving(true);
    try {
      await queryApi(`dashboards/${params.id}/query`, {
        method: "POST",
        body: { content: editorQuery },
      });
      // Clear localStorage after successful save
      editorStorage.clearChanges(params.id);
      dashboard.content = editorQuery;
      router.invalidate();
    } catch (err) {
      if (isRedirect(err)) {
        return navigate(err.options);
      }
      toast({
        title: "Error",
        description: err instanceof Error ? err.message : "An error occurred",
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
      await queryApi(`dashboards/${params.id}/name`, {
        method: "POST",
        body: { name: newName },
      });
      dashboard.name = newName;
      setName(newName);
    } catch (err) {
      if (isRedirect(err)) {
        return navigate(err.options);
      }
      toast({
        title: "Error",
        description: err instanceof Error ? err.message : "An error occurred",
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
        title: "Success",
        description: "Dashboard deleted successfully",
      });
      navigate({ to: "/" });
    } catch (err) {
      if (isRedirect(err)) {
        return navigate(err.options);
      }
      toast({
        title: "Error",
        description: err instanceof Error ? err.message : "An error occurred",
        variant: "error",
      });
    }
  };

  const handleVisibilityChange = async () => {
    try {
      await queryApi(`dashboards/${params.id}/visibility`, {
        method: "POST",
        body: { visibility: selectedVisibility },
      });

      // Save password if visibility is password-protected
      if (selectedVisibility === "password-protected" && password) {
        setSavingPassword(true);
        await queryApi(`dashboards/${params.id}/password`, {
          method: "POST",
          body: { password: password },
        });
        setSavingPassword(false);
      }

      // Show success dialog for password-protected dashboards
      if (selectedVisibility === "password-protected") {
        setShowPasswordSuccessDialog(true);
      } else {
        toast({
          title:
						selectedVisibility === "public"
						  ? "Dashboard made public"
						  : "Dashboard made private",
          description:
						selectedVisibility === "public"
						  ? "Try the link in the sidebar"
						  : "The dashboard is not publicly accessible anymore.",
        });
      }

      dashboard.visibility = selectedVisibility as
				| "public"
				| "private"
				| "password-protected";
      router.invalidate();
    } catch (err) {
      if (isRedirect(err)) {
        return navigate(err.options);
      }
      toast({
        title: "Error",
        description: err instanceof Error ? err.message : "An error occurred",
        variant: "error",
      });
    }
  };

  const handleGeneratePassword = () => {
    setGeneratingPassword(true);
    const newPassword = generatePassword(16);
    setPassword(newPassword);
    // Brief animation feedback
    setTimeout(() => {
      setGeneratingPassword(false);
    }, 400);
  };

  const handleCopyPassword = async () => {
    const success = await copyToClipboard(password);
    if (success) {
      toast({
        title: "Password copied",
        description: "Password copied to clipboard",
      });
    } else {
      toast({
        title: "Error",
        description: "Failed to copy password",
        variant: "error",
      });
    }
  };

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

  const handleOpenVisibilityDialog = () => {
    setSelectedVisibility(dashboard.visibility || "private");
    setPassword("");
    setShowVisibilityDialog(true);
  };

  const handleDiscardChanges = () => {
    editorStorage.clearChanges(params.id);
    setEditorQuery(dashboard.content);
    setRunningQuery(dashboard.content);
    setShowDiscardDialog(false);
  };

  // Load initial preview
  useEffect(() => {
    previewDashboard();
    return () => {
      // Abort any pending preview when unmounting
      if (previewAbortRef.current) {
        previewAbortRef.current.abort();
      }
    };
  }, [previewDashboard]);

  return (
    <MenuProvider currentPath={dashboard.path}>
      <Helmet>
        <title>Edit Dashboard - {dashboard.name}</title>
      </Helmet>

      <div className="h-dvh flex flex-col">
        <div className="h-[42dvh] flex flex-col overflow-y-hidden max-h-[90dvh] min-h-[12dvh] resize-y shrink-0 shadow-sm dark:shadow-none">
          <div className="flex items-center p-2 border-b border-cb dark:border-none">
            <MenuTrigger className="pr-2">
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
              <VariablesMenu onVariablesChange={previewDashboard} />
              {(systemConfig.publicSharingEnabled ||
								systemConfig.passwordProtectedSharingEnabled) && (
                <div className="my-2 px-4">
                  <Button
                    onClick={handleOpenVisibilityDialog}
                    variant="secondary"
                    className="mt-4 capitalize"
                  >
                    {dashboard.visibility === "password-protected"
                      ? "Password Protected"
                      : dashboard.visibility || "private"}
                    <RiArrowDownSLine className="size-4 inline ml-1.5 mt-0.5 fill-ctext2 dark:fill-dtext2" />
                  </Button>
                  {(dashboard.visibility === "public" ||
											dashboard.visibility === "password-protected") && (
                    <PublicLink href={`../../view/${params.id}`} />
                  )}
                </div>
              )}
              <Button
                onClick={() => setShowDeleteDialog(true)}
                variant="destructive"
                className="mt-4 mx-4"
              >
								Delete Dashboard
              </Button>
              {loadDuration && (
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
									Save
                </Button>
              </form>
            ) : (
              <div className="hidden sm:block flex-grow">
                <Tooltip
                  showArrow={false}
                  asChild
                  content="Click to edit dashboard name"
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
							View Dashboard
            </Link>

            <div className="space-x-2">
              <Tooltip showArrow={false} asChild content="Discard Changes">
                <Button
                  onClick={() => setShowDiscardDialog(true)}
                  className={cx("ml-2", {
                    hidden: editorQuery === dashboard.content,
                  })}
                  disabled={editorQuery === dashboard.content}
                  variant="destructive"
                >
									Discard
                </Button>
              </Tooltip>
              <Tooltip showArrow={false} asChild content="Save Dashboard">
                <Button
                  onClick={handleSave}
                  className={cx("ml-2", {
                    hidden: editorQuery === dashboard.content,
                  })}
                  disabled={saving || editorQuery === dashboard.content}
                  isLoading={saving}
                  variant="secondary"
                >
									Save
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
									Run
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
          {previewError && <PreviewError>{previewError}</PreviewError>}
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
            <DialogTitle>Confirm Deletion</DialogTitle>
            <DialogDescription>
              {"Are you sure you want to delete the dashboard \"%%\"?".replace(
                "%%",
                dashboard.name,
              )}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button onClick={() => setShowDeleteDialog(false)}>Cancel</Button>
            <Button
              variant="destructive"
              onClick={() => {
                handleDelete();
                setShowDeleteDialog(false);
              }}
            >
							Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={showDiscardDialog} onOpenChange={setShowDiscardDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Discard Changes</DialogTitle>
            <DialogDescription>
							Are you sure you want to discard your unsaved changes? This action
							cannot be undone.
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

      <Dialog
        open={showVisibilityDialog}
        onOpenChange={setShowVisibilityDialog}
      >
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Dashboard Visibility</DialogTitle>
            <DialogDescription>
							Choose who can access this dashboard
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <div className="space-y-3">
              <label className="flex items-start space-x-3">
                <input
                  type="radio"
                  name="visibility"
                  value="private"
                  checked={selectedVisibility === "private"}
                  onChange={(e) => setSelectedVisibility(e.target.value)}
                  className="mt-1"
                />
                <div>
                  <div className="font-medium">Private</div>
                  <div className="text-sm text-ctext2 dark:text-dtext2">
										Only you can access this dashboard
                  </div>
                </div>
              </label>

              {systemConfig.passwordProtectedSharingEnabled && (
                <>
                  <label className="flex items-start space-x-3">
                    <input
                      type="radio"
                      name="visibility"
                      value="password-protected"
                      checked={selectedVisibility === "password-protected"}
                      onChange={(e) => setSelectedVisibility(e.target.value)}
                      className="mt-1"
                    />
                    <div className="flex-grow">
                      <div className="font-medium">Password Protected</div>
                      <div className="text-sm text-ctext2 dark:text-dtext2">
												Anyone with the link and password can access
                      </div>
                    </div>
                  </label>

                  {selectedVisibility === "password-protected" && (
                    <div className="ml-6 space-y-3">
                      <div className="flex items-center space-x-2">
                        <div className="relative flex-grow">
                          <input
                            type={showPassword ? "text" : "password"}
                            value={password}
                            onChange={(e) => setPassword(e.target.value)}
                            placeholder={
                              dashboard.visibility === "password-protected"
                                ? "Set new password"
                                : "Password"
                            }
                            className={cx(
                              "w-full px-3 py-2 border rounded-md pr-20",
                              "bg-cbgs dark:bg-dbgs border-cb dark:border-db",
                              focusRing,
                            )}
                            minLength={4}
                          />
                          <div className="absolute right-1 top-1 flex space-x-1">
                            <Button
                              type="button"
                              variant="ghost"
                              onClick={() => setShowPassword(!showPassword)}
                              className="p-1.5"
                            >
                              {showPassword ? (
                                <RiEyeOffLine className="size-4" />
                              ) : (
                                <RiEyeLine className="size-4" />
                              )}
                            </Button>
                            {password && (
                              <Button
                                type="button"
                                variant="ghost"
                                onClick={handleCopyPassword}
                                className="p-1.5"
                              >
                                <RiFileCopyLine className="size-4" />
                              </Button>
                            )}
                          </div>
                        </div>
                        <Tooltip
                          showArrow={false}
                          content="Generate a secure 16-character password"
                        >
                          <Button
                            type="button"
                            variant="secondary"
                            onClick={handleGeneratePassword}
                            className="px-3.5 py-2.5"
                            disabled={generatingPassword}
                          >
                            <RiRefreshLine
                              className={cx(
                                "size-5 transition-transform duration-400",
                                generatingPassword && "animate-spin",
                              )}
                            />
                          </Button>
                        </Tooltip>
                      </div>
                      <div className="text-xs text-ctext2 dark:text-dtext2">
												Password must be at least 4 characters long
                      </div>
                      {password && password.length < 4 && (
                        <div className="text-xs text-cerr dark:text-derr">
													Password is too short
                        </div>
                      )}
                    </div>
                  )}
                </>
              )}

              {systemConfig.publicSharingEnabled && (
                <label className="flex items-start space-x-3">
                  <input
                    type="radio"
                    name="visibility"
                    value="public"
                    checked={selectedVisibility === "public"}
                    onChange={(e) => setSelectedVisibility(e.target.value)}
                    className="mt-1"
                  />
                  <div>
                    <div className="font-medium">Public</div>
                    <div className="text-sm text-ctext2 dark:text-dtext2">
											Anyone with the link can access
                    </div>
                  </div>
                </label>
              )}
            </div>
          </div>

          <DialogFooter>
            <Button
              onClick={() => setShowVisibilityDialog(false)}
              variant="secondary"
            >
							Cancel
            </Button>
            <Button
              variant="primary"
              onClick={() => {
                handleVisibilityChange();
                setShowVisibilityDialog(false);
              }}
              disabled={
                selectedVisibility === "password-protected" &&
								password.length < 4
              }
              isLoading={savingPassword}
            >
							Save
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={showPasswordSuccessDialog}
        onOpenChange={setShowPasswordSuccessDialog}
      >
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Dashboard Password Protected</DialogTitle>
            <DialogDescription>
							Your dashboard is now password protected. Copy the password now -
							you won't be able to see it again.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">Password</label>
              <div className="flex items-center space-x-2">
                <div className="relative flex-grow">
                  <input
                    type={showSuccessPassword ? "text" : "password"}
                    value={password}
                    readOnly
                    className={cx(
                      "w-full px-3 py-2 border rounded-md pr-20 bg-gray-50 dark:bg-gray-800 border-cb dark:border-db",
                      "font-mono text-sm",
                    )}
                  />
                  <div className="absolute right-1 top-1 flex space-x-1">
                    <Button
                      type="button"
                      variant="ghost"
                      onClick={() =>
                        setShowSuccessPassword(!showSuccessPassword)
                      }
                      className="p-1.5"
                    >
                      {showSuccessPassword ? (
                        <RiEyeOffLine className="size-4" />
                      ) : (
                        <RiEyeLine className="size-4" />
                      )}
                    </Button>
                    <Button
                      type="button"
                      variant="ghost"
                      onClick={handleCopyPassword}
                      className="p-1.5"
                    >
                      <RiFileCopyLine className="size-4" />
                    </Button>
                  </div>
                </div>
              </div>
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">Share Link</label>
              <div className="flex items-center space-x-2">
                <input
                  type="text"
                  value={`${window.location.origin}${window.shaper.defaultBaseUrl}view/${params.id}`}
                  readOnly
                  className={cx(
                    "flex-grow px-3 py-2 border rounded-md bg-gray-50 dark:bg-gray-800 border-cb dark:border-db",
                    "text-sm",
                  )}
                />
                <Button
                  type="button"
                  variant="secondary"
                  onClick={() => {
                    copyToClipboard(
                      `${window.location.origin}${window.shaper.defaultBaseUrl}view/${params.id}`,
                    );
                    toast({
                      title: "Link copied",
                      description: "Share link copied to clipboard",
                    });
                  }}
                  className="px-3"
                >
									Copy
                </Button>
              </div>
            </div>

            <div className="pt-2">
              <a href={`../../view/${params.id}`} target="_blank">
                <Button type="button" variant="primary" className="w-full">
									Open Shared Dashboard
                  <RiExternalLinkLine className="size-4 ml-2" />
                </Button>
              </a>
            </div>
          </div>

          <DialogFooter>
            <Button
              onClick={() => setShowPasswordSuccessDialog(false)}
              variant="secondary"
            >
							Close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </MenuProvider>
  );
}
