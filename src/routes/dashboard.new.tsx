import { Editor } from "@monaco-editor/react";
import * as monaco from "monaco-editor";
import { z } from "zod";
import {
  createFileRoute,
  isRedirect,
  useNavigate,
} from "@tanstack/react-router";
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
  isMac,
  varsParamSchema,
} from "../lib/utils";
import { translate } from "../lib/translate";
import { editorStorage } from "../lib/editorStorage";
import { Button } from "../components/tremor/Button";
import { useQueryApi } from "../hooks/useQueryApi";
import { Menu } from "../components/Menu";
import { Result } from "../lib/dashboard";
import { useToast } from "../hooks/useToast";
import { Tooltip } from "../components/tremor/Tooltip";

const defaultQuery = "-- Enter your SQL query here";

export const Route = createFileRoute("/dashboard/new")({
  validateSearch: z.object({
    vars: varsParamSchema,
  }),
  component: NewDashboard,
});

function NewDashboard() {
  const { vars } = Route.useSearch();
  const auth = useAuth();
  const queryApi = useQueryApi();
  const navigate = useNavigate({ from: "/dashboard/new" });
  const [editorQuery, setEditorQuery] = useState(defaultQuery);
  const [runningQuery, setRunningQuery] = useState(defaultQuery);
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
      setEditorQuery(unsavedContent);
      setRunningQuery(unsavedContent);
    }
  }, []);

  const previewDashboard = useCallback(async () => {
    setPreviewError(null);
    setIsPreviewLoading(true);
    try {
      const searchParams = getSearchParamString(vars);
      const data = await queryApi(`/api/query/dashboard?${searchParams}`, {
        method: "POST",
        body: {
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
  }, [queryApi, vars, runningQuery, navigate]);

  useEffect(() => {
    previewDashboard();
  }, [previewDashboard]);

  const handleRun = useCallback(() => {
    if (editorQuery !== runningQuery) {
      setRunningQuery(editorQuery);
    } else {
      previewDashboard();
    }
  }, [editorQuery, runningQuery, previewDashboard]);

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

  const handleQueryChange = (value: string | undefined) => {
    const newQuery = value || "";
    // Save to localStorage
    if (newQuery !== defaultQuery && newQuery.trim() !== "") {
      editorStorage.saveChanges("new", newQuery);
    } else {
      editorStorage.clearChanges("new");
    }
    setEditorQuery(newQuery);
  };

  const handleCreate = useCallback(async () => {
    const name = window.prompt(
      `${translate("Enter a name for the dashboard")}:`,
    );
    if (!name) return;

    setCreating(true);
    try {
      const { id } = await queryApi("/api/dashboards", {
        method: "POST",
        body: {
          name,
          content: editorQuery,
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
          err instanceof Error ? err.message : translate("An error occurred"),
        variant: "error",
      });
      setCreating(false);
    }
  }, [queryApi, editorQuery, navigate, vars, toast]);

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
          <div className="flex justify-between items-center p-2 lg:pr-0 border-b">
            <div className="flex items-center space-x-2">
              <Menu isNewPage>
                <div className="mt-6 px-4 w-full">
                  <label>
                    <span className="text-lg font-medium font-display ml-1 mb-2 block">
                      {translate("Variables")}
                    </span>
                    <textarea
                      className={cx(
                        "w-full px-3 py-1.5 bg-cbg dark:bg-dbg text-sm border border-cb dark:border-db shadow-sm outline-none ring-0 rounded-md font-mono resize-none h-32",
                        focusRing,
                        hasVariableError && hasErrorInput,
                      )}
                      onChange={(event) => {
                        onVariablesEdit(event.target.value);
                      }}
                      defaultValue={JSON.stringify(auth.variables, null, 2)}
                    ></textarea>
                  </label>
                </div>
              </Menu>
              <h1 className="text-xl font-semibold font-display px-1 hidden md:block lg:hidden 2xl:block">
                {translate("New Dashboard")}
              </h1>
            </div>
            <div className="space-x-2">
              <Tooltip showArrow={false} asChild content="Create Dashboard">
                <Button
                  onClick={handleCreate}
                  disabled={creating}
                  isLoading={creating}
                  variant="secondary"
                >
                  {translate("Create")}
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
                >
                  {translate("Run")}
                </Button>
              </Tooltip>
            </div>
          </div>

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
