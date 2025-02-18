import { Editor } from "@monaco-editor/react";

import { z } from "zod";
import { createFileRoute, isRedirect, Link, useNavigate } from "@tanstack/react-router";
import { useCallback, useEffect, useState } from "react";
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
import { Menu } from "../components/Menu";
import { useToast } from "../hooks/useToast";

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
  const auth = useAuth();
  const queryApi = useQueryApi();
  const navigate = useNavigate({ from: "/dashboards/$dashboardId/edit" });
  const [query, setQuery] = useState(dashboard.content);
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
    const unsavedContent = editorStorage.getChanges(params.dashboardId);
    if (unsavedContent && unsavedContent !== dashboard.content) {
      if (
        window.confirm(
          "There are unsaved previous edits. Do you want to restore them?",
        )
      ) {
        setQuery(unsavedContent);
      } else {
        editorStorage.clearChanges(params.dashboardId);
      }
    }
  }, [params.dashboardId, dashboard.content]);

  const previewDashboard = useCallback(async () => {
    setPreviewError(null);
    setIsPreviewLoading(true);
    editorStorage.saveChanges(params.dashboardId, query);
    try {
      const searchParams = getSearchParamString(vars);
      const data = await queryApi(`/api/query/dashboard?${searchParams}`, {
        method: "POST",
        body: {
          dashboardId: params.dashboardId,
          content: query,
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
  }, [queryApi, params, vars, query, navigate]);

  const debounceSetQuery = useDebouncedCallback(async (newQuery: string) => {
    return setQuery(newQuery);
  }, 1000);

  // Update textarea onChange handler
  const handleQueryChange = (value: string | undefined) => {
    const newQuery = value || "";
    debounceSetQuery(newQuery);
  };

  const handleSave = useCallback(async () => {
    setSaving(true);
    try {
      await queryApi(
        `/api/dashboards/${params.dashboardId}/query`,
        {
          method: "POST",
          body: { content: query },
        },
      );
      dashboard.content = query;
      // Clear localStorage after successful save
      editorStorage.clearChanges(params.dashboardId);
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
  }, [queryApi, params.dashboardId, query, dashboard, navigate, toast]);

  const handleVarsChanged = useCallback(
    (newVars: any) => {
      navigate({
        replace: true,
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
      await queryApi(`/api/dashboards/${params.dashboardId}`, {
        method: "DELETE",
      });
      // Navigate back to dashboard list
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

  // Load initial preview
  useEffect(() => {
    previewDashboard();
  }, [previewDashboard]);

  return (
    <div className="h-screen flex flex-col">
      <Helmet>
        <title>
          {translate("Edit Dashboard")} - {dashboard.name}
        </title>
      </Helmet>

      <div className="flex-1 flex overflow-hidden">
        <div className="w-full lg:w-1/2 overflow-hidden">
          <div className="flex justify-between items-center p-2 border-b">
            <div className="flex items-center space-x-4">
              <Menu>
                <div className="mt-6 px-4">
                  <label className="block">
                    <p className="text-lg font-medium font-display ml-1 mb-2">
                      {translate("Variables")}
                    </p>
                    <textarea
                      className={cx(
                        "w-full px-3 py-1.5 bg-cbgl dark:bg-dbgl text-sm border border-cb dark:border-db shadow-sm outline-none ring-0 rounded-md font-mono resize-none",
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
                    onClick={handleDelete}
                    variant="destructive"
                    className="mt-4"
                  >
                    {translate("Delete Dashboard")}
                  </Button>
                </div>
              </Menu>
              {editingName ? (
                <form
                  onSubmit={(e) => {
                    e.preventDefault();
                    const input = e.currentTarget.querySelector("input");
                    if (input) {
                      input.blur();
                    }
                  }}
                  className="inline-block"
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
                      "text-2xl font-semibold font-display px-2 py-1 border rounded",
                      "bg-cbga dark:bg-dbga border-cb dark:border-db shadow-sm outline-none ring-0 rounded-md",
                      focusRing,
                    )}
                    autoFocus
                    disabled={savingName}
                  />
                </form>
              ) : (
                <h1
                  className="text-2xl font-semibold font-display cursor-pointer hover:bg-cbga dark:hover:bg-dbga px-2 py-1 rounded hidden md:block lg:hidden 2xl:block"
                  onClick={() => setEditingName(true)}
                  title={translate("Click to edit dashboard name")}
                >
                  {name}
                </h1>
              )}
            </div>
            <div className="space-x-2">
              <Link
                to="/dashboards/$dashboardId"
                params={{ dashboardId: params.dashboardId }}
                search={() => ({ vars })}
                className="px-4 py-2 text-ctext2 dark:text-dtext2 hover:text-ctext dark:hover:text-dtext hover:underline transition-colors duration-200"
              >
                {translate("View Dashboard")}
              </Link>
              <Button
                onClick={handleSave}
                disabled={saving || query === dashboard.content}
                isLoading={saving}
                loadingText={translate("Saving")}
              >
                {translate("Save")}
              </Button>
            </div>
          </div>

          <Editor
            height="100%"
            defaultLanguage="sql"
            value={query}
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
          />
        </div>

        <div className="hidden lg:block w-1/2 overflow-auto relative">
          {previewError && (
            <div className="fixed w-1/2 h-full p-4 z-50 backdrop-blur-sm flex justify-center items-center">
              <div className="p-4 bg-red-100 text-red-700 h-fit rounded">
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
    </div>
  );
}
