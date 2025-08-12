// SPDX-License-Identifier: MPL-2.0

import { createFileRoute, isRedirect, useNavigate } from '@tanstack/react-router'
import { useCallback, useEffect, useState } from 'react'
import { Helmet } from 'react-helmet'
import { RiPencilLine, RiCloseLine } from "@remixicon/react";
import {
  cx,
  focusRing,
  isMac,
} from '../lib/utils'
import { translate } from '../lib/translate'
import { editorStorage } from '../lib/editorStorage'
import { Button } from '../components/tremor/Button'
import { useQueryApi } from '../hooks/useQueryApi'
import { MenuProvider } from '../components/providers/MenuProvider'
import { MenuTrigger } from '../components/MenuTrigger'
import { useToast } from '../hooks/useToast'
import { Tooltip } from '../components/tremor/Tooltip'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '../components/tremor/Dialog'
import { SqlEditor } from "../components/SqlEditor";
import { PreviewError } from "../components/PreviewError";
import { TaskResults, TaskResult } from "../components/TaskResults";
import "../lib/editorInit";

interface TaskData {
  id: string
  name: string
  content: string
}

export const Route = createFileRoute('/tasks/$id')({
  component: TaskEdit,
})

function TaskEdit() {
  const { id } = Route.useParams()
  const queryApi = useQueryApi()
  const navigate = useNavigate()

  const [task, setTask] = useState<TaskData | null>(null)
  const [loading, setLoading] = useState(true)
  const [editorQuery, setEditorQuery] = useState('')
  const [saving, setSaving] = useState(false)
  const [taskData, setTaskData] = useState<TaskResult | undefined>(undefined)
  const [previewError, setPreviewError] = useState<string | null>(null)
  const [isPreviewLoading, setIsPreviewLoading] = useState(false)
  const [showDeleteDialog, setShowDeleteDialog] = useState(false)
  const [editingName, setEditingName] = useState(false)
  const [name, setName] = useState('')
  const [savingName, setSavingName] = useState(false)
  const { toast } = useToast()

  const loadTask = useCallback(async () => {
    try {
      setLoading(true)
      const data = await queryApi(`tasks/${id}`)
      setTask(data)
      setName(data.name)
      setEditorQuery(data.content)
    } catch (err) {
      if (isRedirect(err)) {
        return navigate(err.options)
      }
      toast({
        title: translate('Error'),
        description: err instanceof Error ? err.message : translate('An error occurred'),
        variant: 'error',
      })
    } finally {
      setLoading(false)
    }
  }, [queryApi, id, navigate, toast])

  useEffect(() => {
    loadTask()
  }, [loadTask])

  // Check for unsaved changes when component mounts
  useEffect(() => {
    if (!task) return

    const unsavedContent = editorStorage.getChanges(id)
    if (unsavedContent && unsavedContent !== task.content) {
      setEditorQuery(unsavedContent)
    }
  }, [task, id])

  const handleRun = useCallback(async () => {
    setPreviewError(null)
    setIsPreviewLoading(true)
    try {
      const data = await queryApi('run/task', {
        method: 'POST',
        body: {
          content: editorQuery,
        },
      })
      setTaskData(data)
    } catch (err) {
      if (isRedirect(err)) {
        return navigate(err.options)
      }
      setPreviewError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setIsPreviewLoading(false)
    }
  }, [queryApi, editorQuery, navigate])

  const handleQueryChange = useCallback((value: string | undefined) => {
    const newQuery = value || ''

    // Save to localStorage
    if (task && newQuery !== task.content && newQuery.trim() !== '') {
      editorStorage.saveChanges(id, newQuery)
    } else {
      editorStorage.clearChanges(id)
    }
    setEditorQuery(newQuery)
  }, [task, id])

  const handleSave = useCallback(async () => {
    if (!task) return

    setSaving(true)
    try {
      await queryApi(`tasks/${id}/content`, {
        method: 'POST',
        body: {
          content: editorQuery,
        },
      })

      // Clear localStorage after successful save
      editorStorage.clearChanges(id)

      // Update local state
      setTask(prev => prev ? { ...prev, content: editorQuery } : null)

      toast({
        title: translate('Success'),
        description: translate('Task saved successfully'),
        variant: 'success',
      })
    } catch (err) {
      if (isRedirect(err)) {
        return navigate(err.options)
      }
      toast({
        title: translate('Error'),
        description: err instanceof Error ? err.message : translate('An error occurred'),
        variant: 'error',
      })
    } finally {
      setSaving(false)
    }
  }, [queryApi, id, editorQuery, task, navigate, toast])

  const handleSaveName = async (newName: string) => {
    if (!task || newName === task.name) {
      setEditingName(false);
      return;
    }
    setSavingName(true);
    try {
      await queryApi(`tasks/${id}/name`, {
        method: 'POST',
        body: { name: newName },
      });
      task.name = newName;
			setName(newName);
		} catch (err) {
			if (isRedirect(err)) {
				return navigate(err.options);
			}
			toast({
				title: translate('Error'),
				description: err instanceof Error ? err.message : translate('An error occurred'),
				variant: 'error',
			})
			// Revert name on error
			setName(task.name);
		} finally {
			setSavingName(false);
			setEditingName(false);
		}
	}

	const handleDelete = async () => {
		try {
			await queryApi(`tasks/${id}`, {
				method: 'DELETE',
			})
			// Navigate back to task list
			toast({
				title: translate('Success'),
				description: translate('Task deleted successfully'),
			})
			navigate({ to: '/' });
		} catch (err) {
			if (isRedirect(err)) {
				return navigate(err.options);
			}
			toast({
				title: translate('Error'),
				description: err instanceof Error ? err.message : translate('An error occurred'),
				variant: 'error',
			})
		}
	}

	if (loading) {
		return (
			<div className="h-dvh flex items-center justify-center">
				<div className="text-center">
					<div className="animate-spin h-8 w-8 border-2 border-cb dark:border-db border-t-cprimary dark:border-t-dprimary rounded-full mx-auto mb-4"></div>
					<p className="text-ctext2 dark:text-dtext2">{translate('Loading task...')}</p>
				</div>
			</div>
		)
	}

	if (!task) {
		return (
			<div className="h-dvh flex items-center justify-center">
				<div className="text-center">
					<p className="text-ctext2 dark:text-dtext2 text-lg">{translate('Task not found')}</p>
				</div>
			</div>
		)
	}

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
								<Button
									onClick={() => setShowDeleteDialog(true)}
									variant='destructive'
									className='mt-4'
								>
									{translate('Delete Task')}
								</Button>
							</div>
						</MenuTrigger>

						{editingName ? (
							<form
								onSubmit={(e) => {
									e.preventDefault()
									const input = e.currentTarget.querySelector("input");
									if (input && !savingName) {
										handleSaveName(name)
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
										e.preventDefault()
										setEditingName(false)
										setName(task.name)
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
									{translate('Save')}
								</Button>
							</form>
						) : (
							<div className="hidden sm:block flex-grow">
								<Tooltip
									showArrow={false}
									asChild
									content={translate("Click to edit task name")}
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
								content='Save Task'
							>
								<Button
									onClick={handleSave}
									className={cx("ml-2", { "hidden": editorQuery === task?.content })}
									disabled={saving || editorQuery === task?.content}
									isLoading={saving}
									variant='secondary'
								>
									{translate('Save')}
								</Button>
							</Tooltip>
							<Tooltip
								showArrow={false}
								asChild
								content={`Press ${isMac() ? '⌘' : 'Ctrl'} + Enter to run`}
							>
								<Button
									onClick={handleRun}
									disabled={isPreviewLoading}
									isLoading={isPreviewLoading}
								>
									{translate('Run')}
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
						<DialogTitle>{translate("Confirm Deletion")}</DialogTitle>
						<DialogDescription>
							{translate('Are you sure you want to delete the task "%%"?').replace(
								'%%',
								task.name,
							)}
						</DialogDescription>
					</DialogHeader>
					<DialogFooter>
						<Button onClick={() => setShowDeleteDialog(false)}>
							{translate('Cancel')}
						</Button>
						<Button
							variant='destructive'
							onClick={() => {
								handleDelete()
								setShowDeleteDialog(false);
							}}
						>
							{translate('Delete')}
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>
		</MenuProvider>
	)
}