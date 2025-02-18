import { Editor } from "@monaco-editor/react";
import { z } from "zod";
import { createFileRoute, isRedirect, useNavigate } from "@tanstack/react-router";
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
import { Button } from "../components/tremor/Button";
import { useQueryApi } from "../hooks/useQueryApi";
import { Menu } from "../components/Menu";
import { Result } from "../lib/dashboard";
import { useToast } from "../hooks/useToast";

const defaultQuery = "-- Enter your SQL query here"

export const Route = createFileRoute("/dashboard/new")({
  validateSearch: z.object({
    vars: varsParamSchema,
  }),
  component: NewDashboard,
});

function NewDashboard() {
  const { vars } = Route.useSearch();
  const auth = useAuth();
  const queryApi = useQueryApi()
  const navigate = useNavigate({ from: "/dashboard/new" });
  const [query, setQuery] = useState(defaultQuery);
  const [creating, setCreating] = useState(false);
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
    const unsavedContent = editorStorage.getChanges("new");
    if (unsavedContent) {
      setQuery(unsavedContent);
    }
  }, []);

  const previewDashboard = useCallback(async () => {
    setPreviewError(null);
    setIsPreviewLoading(true);
    // Save to localStorage
    if (query !== defaultQuery && query.trim() !== "") {
      editorStorage.saveChanges("new", query);
    } else {
      editorStorage.clearChanges("new");
    }
    try {
      const searchParams = getSearchParamString(vars);
      const data = await queryApi(`/api/query/dashboard?${searchParams}`, {
        method: "POST",
        body: {
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
  }, [queryApi, vars, query, navigate]);

  const debounceSetQuery = useDebouncedCallback(async (newQuery: string) => {
    return setQuery(newQuery);
  }, 1000);

  useEffect(() => {
    previewDashboard();
  }, [previewDashboard]);

  const handleQueryChange = (value: string | undefined) => {
    const newQuery = value || "";
    debounceSetQuery(newQuery);
  };

  const handleCreate = useCallback(async () => {
    const name = window.prompt(`${translate("Enter a name for the dashboard")}:`);
    if (!name) return;

    setCreating(true);
    try {
      const { id } = await queryApi("/api/dashboards", {
        method: "POST",
        body: {
          name,
          content: query,
        },
      });
      // Clear localStorage after successful save
      editorStorage.clearChanges("new");

      // Navigate to the edit page of the new dashboard
      navigate({
        replace: true,
        to: "/dashboards/$dashboardId/edit",
        params: { dashboardId: id },
        search: () => ({ vars }),
      });
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
      setCreating(false);
    }
  }, [queryApi, query, navigate, vars, toast]);

  const handleVarsChanged = useCallback(
    (newVars: any) => {
      navigate({
        replace: true,
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
          previewDashboard();
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

      <div className="flex-1 flex overflow-hidden">
        <div className="w-full lg:w-1/2 overflow-hidden">
          <div className="flex justify-between items-center py-4 pl-2 pr-4 border-b">
            <div className="flex items-center space-x-4">
              <Menu isNewPage>
                <div className="mt-6 px-4 w-full">
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
              </Menu>
            </div>
            <div className="space-x-2">
              <Button
                onClick={handleCreate}
                disabled={creating}
                isLoading={creating}
                loadingText={translate("Creating")}
              >
                {translate("Create")}
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
