import { Editor } from "@monaco-editor/react";
import { loader } from "@monaco-editor/react";
import * as monaco from "monaco-editor";
import editorWorker from "monaco-editor/esm/vs/editor/editor.worker?worker";

import { z } from "zod";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useCallback, useEffect, useState } from "react";
import { Helmet } from "react-helmet";
import { useAuth } from "../lib/auth";
import { Dashboard } from "../components/dashboard";
import { RiCloseLargeLine, RiMenuLine } from "@remixicon/react";
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

self.MonacoEnvironment = {
  getWorker() {
    return new editorWorker();
  },
};
loader.config({ monaco });
loader.init();

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
      auth: { getJwt },
    },
  }) => {
    const jwt = await getJwt();
    const response = await fetch(`/api/dashboards/${dashboardId}/query`, {
      headers: {
        Authorization: jwt,
      },
    });
    if (!response.ok) {
      throw new Error("Failed to load dashboard query");
    }
    const data = await response.json();
    return data as IDashboard;
  },
  component: DashboardEditor,
});

function DashboardEditor() {
  const params = Route.useParams();
  const { vars } = Route.useSearch();
  const dashboard = Route.useLoaderData();
  const auth = useAuth();
  const navigate = useNavigate({ from: "/dashboards/$dashboardId/edit" });
  const [query, setQuery] = useState(dashboard.content);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const [hasVariableError, setHasVariableError] = useState(false);
  const [previewData, setPreviewData] = useState<Result | null>(null);
  const [previewError, setPreviewError] = useState<string | null>(null);
  const [isPreviewLoading, setIsPreviewLoading] = useState(false);
  const [isDarkMode, setIsDarkMode] = useState(
    window.matchMedia("(prefers-color-scheme: dark)").matches,
  );

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

  // Add debounced preview function
  const previewDashboard = useDebouncedCallback(async (newQuery: string) => {
    setPreviewError(null);
    setIsPreviewLoading(true);
    editorStorage.saveChanges(params.dashboardId, newQuery);
    try {
      const jwt = await auth.getJwt();
      const searchParams = getSearchParamString(vars);
      const response = await fetch(`/api/query/dashboard?${searchParams}`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: jwt,
        },
        body: JSON.stringify({
          dashboardId: params.dashboardId,
          content: newQuery,
        }),
      });

      if (!response.ok) {
        const data = await response.json();
        throw new Error(data.error || "Failed to preview dashboard");
      }

      const data = await response.json();
      setPreviewData(data);
    } catch (err) {
      setPreviewError(err instanceof Error ? err.message : "Unknown error");
    } finally {
      setIsPreviewLoading(false);
    }
  }, 1000);

  // Update textarea onChange handler
  const handleQueryChange = (value: string | undefined) => {
    const newQuery = value || "";
    setQuery(newQuery);
    previewDashboard(newQuery);
  };

  const handleSave = useCallback(async () => {
    setSaving(true);
    setError(null);
    try {
      const jwt = await auth.getJwt();
      const response = await fetch(
        `/api/dashboards/${params.dashboardId}/query`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: jwt,
          },
          body: JSON.stringify({ content: query }),
        },
      );

      if (!response.ok) {
        throw new Error("Failed to save dashboard query");
      }
      dashboard.content = query;
      // Clear localStorage after successful save
      editorStorage.clearChanges(params.dashboardId);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown error");
    } finally {
      setSaving(false);
    }
  }, [auth, params.dashboardId, query, dashboard]);

  const handleDashboardError = useCallback((err: Error) => {
    setPreviewError(err.message);
  }, []);

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

  const onVariablesEdit = useDebouncedCallback((value) => {
    auth.updateVariables(value).then(
      (ok) => {
        setHasVariableError(!ok);
        if (ok) {
          // Refresh preview when variables change
          previewDashboard(query);
        }
      },
      () => {
        setHasVariableError(true);
      },
    );
  }, 500);

  const handleDelete = async () => {
    if (
      !window.confirm(
        `Are you sure you want to delete dashboard "${dashboard.name}"?`,
      )
    ) {
      return;
    }

    try {
      const jwt = await auth.getJwt();
      const response = await fetch(`/api/dashboards/${params.dashboardId}`, {
        method: "DELETE",
        headers: {
          Authorization: jwt,
        },
      });

      if (!response.ok) {
        throw new Error("Failed to delete dashboard");
      }

      // Navigate back to dashboard list
      navigate({ to: "/" });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown error");
    }
  };

  // Load initial preview
  useEffect(() => {
    previewDashboard(query);
  }, [previewDashboard, query, vars]); // Add vars to dependency array

  return (
    <div className="h-screen flex flex-col">
      <Helmet>
        <title>Edit {dashboard.name}</title>
      </Helmet>

      {error && (
        <div className="m-4 p-4 bg-red-100 text-red-700 rounded">{error}</div>
      )}

      <div className="flex-1 flex overflow-hidden">
        <div className="w-1/2 overflow-hidden">
          <div className="flex justify-between items-center p-4 border-b">
            <div className="flex items-center space-x-4">
              <button className="px-1" onClick={() => setIsMenuOpen(true)}>
                <RiMenuLine className="py-1 size-7 text-ctext2 dark:text-dtext2 hover:text-ctext hover:dark:text-dtext transition-colors" />
              </button>
              <Link to="/" className="text-gray-600 hover:text-gray-800">
                ← Overview
              </Link>
              <h1 className="text-2xl font-bold">{dashboard.name}</h1>
            </div>
            <div className="space-x-2">
              <Link
                to="/dashboards/$dashboardId"
                params={{ dashboardId: params.dashboardId }}
                search={() => ({ vars })}
                className="px-4 py-2 text-gray-600 hover:text-gray-800"
              >
                View
              </Link>
              <button
                onClick={handleSave}
                disabled={saving || query === dashboard.content}
                className="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 disabled:opacity-50"
              >
                {saving ? "Saving..." : "Save"}
              </button>
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

        <div className="w-1/2 overflow-auto relative">
          {previewError && (
            <div className="fixed w-1/2 h-full p-4 z-50 backdrop-blur-sm flex justify-center items-center">
              <div className="p-4 bg-red-100 text-red-700 h-fit rounded">
                {previewError}
              </div>
            </div>
          )}
          {isPreviewLoading && (
            <div className="absolute inset-0 bg-white bg-opacity-50 flex items-center justify-center">
              <div className="text-gray-500">Loading preview...</div>
            </div>
          )}
          {previewData && (
            <Dashboard
              id={params.dashboardId}
              vars={vars}
              hash={auth.hash}
              getJwt={auth.getJwt}
              onVarsChanged={handleVarsChanged}
              onError={handleDashboardError}
              data={previewData} // Pass preview data directly to Dashboard
            />
          )}
        </div>
      </div>

      {/* Variables Menu */}
      <div
        className={cx(
          "fixed top-0 h-dvh w-full @sm:w-fit bg-cbga dark:bg-dbga shadow-xl ease-in-out delay-75 duration-300 z-40",
          {
            "-translate-x-[calc(100vw+50px)]": !isMenuOpen,
          },
        )}
      >
        <button onClick={() => setIsMenuOpen(false)}>
          <RiCloseLargeLine className="pl-1 py-1 ml-2 mt-2 size-7 text-ctext2 dark:text-dtext2 hover:text-ctext hover:dark:text-dtext transition-colors" />
        </button>
        <button
          onClick={handleDelete}
          className="px-4 py-2 bg-red-500 text-white rounded hover:bg-red-600"
        >
          Delete
        </button>
        <div className="mt-6 px-5 w-full @sm:w-96">
          <label>
            <span className="text-lg font-medium font-display ml-1 mb-2 block">
              {translate("Variables")}
            </span>
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
        </div>
      </div>
    </div>
  );
}
