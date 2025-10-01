// SPDX-License-Identifier: MPL-2.0

import { createFileRoute, isRedirect, useNavigate, useRouter } from "@tanstack/react-router";
import { useCallback, useState, useEffect } from "react";
import { Helmet } from "react-helmet";
import { RiPencilLine, RiCloseLine } from "@remixicon/react";
import {
  cx,
  focusRing,
  isMac,
} from "../lib/utils";
import { editorStorage } from "../lib/editorStorage";
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
import { SqlEditor } from "../components/SqlEditor";
import { PreviewError } from "../components/PreviewError";
import { TaskResults, TaskResult } from "../components/TaskResults";
import { RelativeDate } from "../components/RelativeDate";
import "../lib/editorInit";

interface TaskData {
  id: string
  name: string
  content: string
  nextRunAt?: string
  lastRunAt?: string
  lastRunSuccess?: boolean
  lastRunDuration?: number
}

export const Route = createFileRoute("/tasks/$id")({
  loader: async ({ context: { queryApi }, params: { id } }) => {
    return queryApi(`tasks/${id}`) as Promise<TaskData>;
  },
  component: TaskEdit,
});

function TaskEdit () {
  const { id } = Route.useParams();
  const task = Route.useLoaderData();
  const queryApi = useQueryApi();
  const navigate = useNavigate();
  const router = useRouter();
  const [editorQuery, setEditorQuery] = useState("");
  const [saving, setSaving] = useState(false);
  const [taskData, setTaskData] = useState<TaskResult | undefined>(undefined);
  const [previewError, setPreviewError] = useState<string | null>(null);
  const [isPreviewLoading, setIsPreviewLoading] = useState(false);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [showDiscardDialog, setShowDiscardDialog] = useState(false);
  const [editingName, setEditingName] = useState(false);
  const [name, setName] = useState(task.name);
  const [savingName, setSavingName] = useState(false);
  const { toast } = useToast();

  // Check for unsaved changes when component mounts
  useEffect(() => {
    const unsavedContent = editorStorage.getChanges(id, "task");
    if (unsavedContent && unsavedContent !== task.content) {
      setEditorQuery(unsavedContent);
    } else {
      setEditorQuery(task.content);
    }
  }, [task, id]);

  const handleRun = useCallback(async () => {
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

  const handleQueryChange = useCallback((value: string | undefined) => {
    const newQuery = value || "";

    // Save to localStorage
    if (task && newQuery !== task.content && newQuery.trim() !== "") {
      editorStorage.saveChanges(id, newQuery, "task");
    } else {
      editorStorage.clearChanges(id, "task");
    }
    setEditorQuery(newQuery);
  }, [task, id]);

  const handleSave = useCallback(async () => {
    setSaving(true);
    try {
      await queryApi(`tasks/${id}/content`, {
        method: "POST",
        body: {
          content: editorQuery,
        },
      });

      // Clear localStorage after successful save
      editorStorage.clearChanges(id, "task");

      toast({
        title: "Success",
        description: "Task saved successfully",
        variant: "success",
      });

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
  }, [queryApi, id, editorQuery, navigate, toast, router]);

  const handleSaveName = async (newName: string) => {
    if (newName === task.name) {
      setEditingName(false);
      return;
    }
    setSavingName(true);
    try {
      await queryApi(`tasks/${id}/name`, {
        method: "POST",
        body: { name: newName },
      });
      setName(newName);
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
      // Revert name on error
      setName(task.name);
    } finally {
      setSavingName(false);
      setEditingName(false);
    }
  };

  const handleDelete = async () => {
    try {
      await queryApi(`tasks/${id}`, {
        method: "DELETE",
      });
      // Navigate back to task list
      toast({
        title: "Success",
        description: "Task deleted successfully",
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

  const handleDiscardChanges = () => {
    editorStorage.clearChanges(id, "task");
    setEditorQuery(task.content);
    setShowDiscardDialog(false);
  };

  return (
    <MenuProvider>
      <Helmet>
        <title>{task.name}</title>
      </Helmet>

      <div className="h-dvh flex flex-col">
        <div className="h-[42dvh] flex flex-col overflow-y-hidden max-h-[90dvh] min-h-[12dvh] resize-y shrink-0 shadow-sm dark:shadow-none">
          <div className="flex items-center p-2 border-b border-cb dark:border-none">
            <MenuTrigger className="pr-2">
              <div className="px-4">
                {task.nextRunAt && (
                  <div className="mt-4 mb-4 text-sm text-ctext2 dark:text-dtext2">
                    <div className="font-medium">Task Scheduled</div>
                    <div><RelativeDate refresh date={new Date(task.nextRunAt)} /></div>
                  </div>
                )}
                {task.lastRunAt && (
                  <div className="mt-4 mb-4 text-sm text-ctext2 dark:text-dtext2">
                    <div className={`font-medium ${task.lastRunSuccess === false ? "text-cerr" : ""}`}>
                      {task.lastRunSuccess === false ? "Last Run Failed" : "Last Run"}
                    </div>
                    <div className={task.lastRunSuccess === false ? "text-cerr" : ""}>
                      <RelativeDate refresh date={new Date(task.lastRunAt)} /> ({task.lastRunDuration}ms)
                    </div>
                  </div>
                )}
                <Button
                  onClick={() => setShowDeleteDialog(true)}
                  variant='destructive'
                  className='mt-4'
                >
                  Delete Task
                </Button>
              </div>
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
                  type='text'
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
                  variant='destructive'
                  type='reset'
                  className='ml-2'
                  onClick={(e) => {
                    e.preventDefault();
                    setEditingName(false);
                    setName(task.name);
                  }}
                >
                  <RiCloseLine className="size-5" />
                </Button>
                <Button
                  type='submit'
                  variant='primary'
                  className='inline ml-2'
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
                  content="Click to edit task name"
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

            <div className="space-x-2">
              <Tooltip
                showArrow={false}
                asChild
                content='Discard Changes'
              >
                <Button
                  onClick={() => setShowDiscardDialog(true)}
                  className={cx("ml-2", { "hidden": editorQuery === task?.content })}
                  disabled={editorQuery === task?.content}
                  variant='destructive'
                >
                  Discard
                </Button>
              </Tooltip>
              <Tooltip
                showArrow={false}
                asChild
                content='Save Task'
              >
                <Button
                  onClick={handleSave}
                  className={cx("ml-2", { "hidden": editorQuery === task?.content })}
                  disabled={saving || editorQuery === task?.content}
                  isLoading={saving}
                  variant='secondary'
                >
                  Save
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
          {previewError && (
            <PreviewError>{previewError}</PreviewError>
          )}
          <TaskResults
            data={taskData}
            loading={isPreviewLoading}
          />
        </div>
      </div>

      <Dialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Confirm Deletion</DialogTitle>
            <DialogDescription>
              {"Are you sure you want to delete the task \"%%\"?".replace(
                "%%",
                task.name,
              )}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button onClick={() => setShowDeleteDialog(false)}>
              Cancel
            </Button>
            <Button
              variant='destructive'
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
              Are you sure you want to discard your unsaved changes? This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button onClick={() => setShowDiscardDialog(false)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleDiscardChanges}
            >
              Discard
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </MenuProvider>
  );
}
