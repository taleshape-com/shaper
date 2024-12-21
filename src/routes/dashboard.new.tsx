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
import { cx, focusRing, hasErrorInput, varsParamSchema } from "../lib/utils";
import { translate } from "../lib/translate";
import { editorStorage } from "../lib/editorStorage";

self.MonacoEnvironment = {
  getWorker() {
    return new editorWorker();
  },
};
loader.config({ monaco });
loader.init();

export const Route = createFileRoute("/dashboard/new")({
  validateSearch: z.object({
    vars: varsParamSchema,
  }),
  component: NewDashboard,
});

function NewDashboard() {
  const { vars } = Route.useSearch();
  const auth = useAuth();
  const navigate = useNavigate({ from: "/dashboard/new" });
  const [query, setQuery] = useState("-- Enter your SQL query here");
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const [hasVariableError, setHasVariableError] = useState(false);
  const [previewData, setPreviewData] = useState<any>(null);
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
    const unsavedContent = editorStorage.getChanges("new");
    if (unsavedContent) {
      if (
        window.confirm(
          "There are unsaved previous edits. Do you want to restore them?",
        )
      ) {
        setQuery(unsavedContent);
      } else {
        editorStorage.clearChanges("new");
      }
    }
  }, []);

  const previewDashboard = useDebouncedCallback(async (newQuery: string) => {
    setPreviewError(null);
    setIsPreviewLoading(true);
    // Save to localStorage
    editorStorage.saveChanges("new", newQuery);
    try {
      const jwt = await auth.getJwt();
      const response = await fetch("/api/query/dashboard", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: jwt,
        },
        body: JSON.stringify({
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

  const handleQueryChange = (value: string | undefined) => {
    const newQuery = value || "";
    setQuery(newQuery);
    previewDashboard(newQuery);
  };

  const handleCreate = useCallback(async () => {
    const title = window.prompt("Enter a title for the dashboard:");
    if (!title) return;

    setCreating(true);
    setError(null);
    try {
      const jwt = await auth.getJwt();
      const response = await fetch("/api/dashboards", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: jwt,
        },
        body: JSON.stringify({
          id: title,
          content: query,
        }),
      });

      if (!response.ok) {
        throw new Error("Failed to create dashboard");
      }

      // Clear localStorage after successful creation
      editorStorage.clearChanges("new");

      // Navigate to the edit page of the new dashboard
      navigate({
        to: "/dashboards/$dashboardId/edit",
        params: { dashboardId: title },
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown error");
    } finally {
      setCreating(false);
    }
  }, [auth, query, navigate]);

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

  return (
    <div className="h-screen flex flex-col">
      <Helmet>
        <title>New Dashboard</title>
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
              <h1 className="text-2xl font-bold">New Dashboard</h1>
            </div>
            <div className="space-x-2">
              <button
                onClick={handleCreate}
                disabled={creating}
                className="px-4 py-2 bg-green-500 text-white rounded hover:bg-green-600 disabled:opacity-50"
              >
                {creating ? "Creating..." : "Create"}
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
              scrollBeyondLastLine: false,
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
              id={"New"}
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
          "fixed top-0 h-dvh w-full sm:w-fit bg-cbga dark:bg-dbga shadow-xl ease-in-out delay-75 duration-300 z-40",
          {
            "-translate-x-[calc(100vw+50px)]": !isMenuOpen,
          },
        )}
      >
        <button onClick={() => setIsMenuOpen(false)}>
          <RiCloseLargeLine className="pl-1 py-1 ml-2 mt-2 size-7 text-ctext2 dark:text-dtext2 hover:text-ctext hover:dark:text-dtext transition-colors" />
        </button>
        <div className="mt-6 px-5 w-full sm:w-96">
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
