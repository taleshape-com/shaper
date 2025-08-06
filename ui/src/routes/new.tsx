// SPDX-License-Identifier: MPL-2.0

import { z } from 'zod'
import {
  createFileRoute,
  isRedirect,
  useNavigate,
} from '@tanstack/react-router'
import { useCallback, useEffect, useState } from 'react'
import { Helmet } from 'react-helmet'
import { useAuth } from '../lib/auth'
import { Dashboard } from '../components/dashboard'
import {
  getSearchParamString,
  isMac,
  varsParamSchema,
} from '../lib/utils'
import { translate } from '../lib/translate'
import { editorStorage } from '../lib/editorStorage'
import { Button } from '../components/tremor/Button'
import { useQueryApi } from '../hooks/useQueryApi'
import { MenuProvider } from '../components/providers/MenuProvider'
import { MenuTrigger } from '../components/MenuTrigger'
import { Result } from '../lib/dashboard'
import { useToast } from '../hooks/useToast'
import { Tooltip } from '../components/tremor/Tooltip'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '../components/tremor/Dialog'
import { Input } from '../components/tremor/Input'
import { VariablesMenu } from '../components/VariablesMenu'
import { SqlEditor } from "../components/SqlEditor";
import { PreviewError } from "../components/PreviewError";
import { WorkflowResults, WorkflowResult } from "../components/WorkflowResults";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../components/tremor/Select'
import "../lib/editorInit";

const defaultQuery = `SELECT 'Dashboard Title'::SECTION;

SELECT 'Label'::LABEL;
SELECT 'Hello World';`;

const defaultWorkflowQuery = `-- Example workflow: Load and process data
CREATE TABLE temp_data AS 
SELECT 1 as id, 'Item A' as name
UNION ALL 
SELECT 2 as id, 'Item B' as name;

SELECT * FROM temp_data;

DROP TABLE temp_data;`;

export const Route = createFileRoute('/new')({
  validateSearch: z.object({
    vars: varsParamSchema,
    type: z.enum(['dashboard', 'workflow']).optional(),
  }),
  component: NewDashboard,
})

function NewDashboard() {
  const { vars, type } = Route.useSearch()
  const auth = useAuth()
  const queryApi = useQueryApi()
  const navigate = useNavigate({ from: '/new' })
  const appType = type || 'dashboard'
  const [editorQuery, setEditorQuery] = useState(appType === 'workflow' ? defaultWorkflowQuery : defaultQuery)
  const [runningQuery, setRunningQuery] = useState(appType === 'workflow' ? defaultWorkflowQuery : defaultQuery)
  const [creating, setCreating] = useState(false)
  const [previewData, setPreviewData] = useState<Result | undefined>(undefined)
  const [workflowData, setWorkflowData] = useState<WorkflowResult | undefined>(undefined)
  const [previewError, setPreviewError] = useState<string | null>(null)
  const [isPreviewLoading, setIsPreviewLoading] = useState(false)
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [dashboardName, setDashboardName] = useState('')
  const { toast } = useToast()

  // Check for unsaved changes when component mounts or type changes
  useEffect(() => {
    const unsavedContent = editorStorage.getChanges('new')
    if (unsavedContent) {
      setEditorQuery(unsavedContent)
      setRunningQuery(unsavedContent)
    } else {
      // Set default content based on app type
      const defaultContent = appType === 'workflow' ? defaultWorkflowQuery : defaultQuery
      setEditorQuery(defaultContent)
      setRunningQuery(defaultContent)
    }
  }, [appType])

  const previewDashboard = useCallback(async () => {
    setPreviewError(null)
    setIsPreviewLoading(true)
    try {
      const searchParams = getSearchParamString(vars)
      const data = await queryApi(`run/dashboard?${searchParams}`, {
        method: 'POST',
        body: {
          content: runningQuery,
        },
      })
      setPreviewData(data)
    } catch (err) {
      if (isRedirect(err)) {
        return navigate(err.options)
      }
      setPreviewError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setIsPreviewLoading(false)
    }
  }, [queryApi, vars, runningQuery, navigate])

  const runWorkflow = useCallback(async () => {
    setPreviewError(null)
    setIsPreviewLoading(true)
    try {
      const data = await queryApi('run/workflow', {
        method: 'POST',
        body: {
          content: runningQuery,
        },
      })
      setWorkflowData(data)
    } catch (err) {
      if (isRedirect(err)) {
        return navigate(err.options)
      }
      setPreviewError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setIsPreviewLoading(false)
    }
  }, [queryApi, runningQuery, navigate])

  useEffect(() => {
    if (appType === 'dashboard') {
      previewDashboard()
    }
  }, [previewDashboard, appType])

  const handleRun = useCallback(() => {
    if (isPreviewLoading) {
      return;
    }
    if (editorQuery !== runningQuery) {
      setRunningQuery(editorQuery)
    } else {
      if (appType === 'workflow') {
        runWorkflow()
      } else {
        previewDashboard()
      }
    }
  }, [editorQuery, runningQuery, previewDashboard, runWorkflow, isPreviewLoading, appType])

  const handleTypeChange = useCallback((newType: string) => {
    navigate({
      search: (old: any) => ({
        ...old,
        type: newType === 'dashboard' ? undefined : newType,
      }),
    })
    
    // Clear results when switching types
    setPreviewData(undefined)
    setWorkflowData(undefined)
    setPreviewError(null)
    
    // Auto-run dashboard when switching to it
    if (newType === 'dashboard') {
      setTimeout(() => previewDashboard(), 0)
    }
  }, [navigate, previewDashboard])

  const handleQueryChange = (value: string | undefined) => {
    const newQuery = value || ''
    const currentDefaultQuery = appType === 'workflow' ? defaultWorkflowQuery : defaultQuery
    
    // Save to localStorage
    if (newQuery !== currentDefaultQuery && newQuery.trim() !== '') {
      editorStorage.saveChanges('new', newQuery)
    } else {
      editorStorage.clearChanges('new')
    }
    setEditorQuery(newQuery)
  }

  const handleCreate = useCallback(async () => {
    if (!dashboardName.trim()) {
      return
    }

    setCreating(true)
    try {
      const { id } = await queryApi('dashboards', {
        method: 'POST',
        body: {
          name: dashboardName,
          content: editorQuery,
        },
      })
      // Clear localStorage after successful save
      editorStorage.clearChanges('new')

      // Navigate to the edit page of the new dashboard
      navigate({
        replace: true,
        to: '/dashboards/$dashboardId/edit',
        params: { dashboardId: id },
        search: () => ({ vars }),
      })
    } catch (err) {
      if (isRedirect(err)) {
        return navigate(err.options)
      }
      toast({
        title: translate('Error'),
        description:
          err instanceof Error ? err.message : translate('An error occurred'),
        variant: 'error',
      })
      setCreating(false)
      setShowCreateDialog(false)
    }
  }, [queryApi, editorQuery, navigate, vars, toast, dashboardName])

  const handleVarsChanged = useCallback(
    (newVars: any) => {
      navigate({
        search: (old: any) => ({
          ...old,
          vars: newVars,
        }),
      })
    },
    [navigate],
  )


  return (
    <MenuProvider isNewPage>
      <Helmet>
        <title>{appType === 'workflow' ? 'New Workflow' : 'New Dashboard'}</title>
      </Helmet>

      <div className="h-dvh flex flex-col">
        <div className="h-[42dvh] flex flex-col overflow-y-hidden max-h-[90dvh] min-h-[12dvh] resize-y shrink-0 shadow-sm dark:shadow-none">
          <div className="flex items-center p-2 border-b border-cb dark:border-none">
            <MenuTrigger className="pr-2">
              {appType === 'dashboard' && (
                <VariablesMenu onVariablesChange={previewDashboard} />
              )}
            </MenuTrigger>

            <div className="flex items-center gap-3 flex-grow">
              <h1 className="text-xl font-semibold font-display">
                {translate('New')}
              </h1>
              <Select value={appType} onValueChange={handleTypeChange}>
                <SelectTrigger className="w-32">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="dashboard">{translate('Dashboard')}</SelectItem>
                  <SelectItem value="workflow">{translate('Workflow')}</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-x-2">
              <Tooltip showArrow={false} asChild content={`Create ${appType === 'workflow' ? 'Workflow' : 'Dashboard'}`}>
                <Button
                  onClick={() => setShowCreateDialog(true)}
                  disabled={creating}
                  variant="secondary"
                >
                  {translate('Create')}
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
          {appType === 'dashboard' ? (
            <Dashboard
              vars={vars}
              hash={auth.hash}
              getJwt={auth.getJwt}
              onVarsChanged={handleVarsChanged}
              data={previewData}
              loading={isPreviewLoading}
            />
          ) : (
            <WorkflowResults
              data={workflowData}
              loading={isPreviewLoading}
            />
          )}
        </div>
      </div>

      <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {translate(`Create ${appType === 'workflow' ? 'Workflow' : 'Dashboard'}`)}
            </DialogTitle>
          </DialogHeader>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              handleCreate();
            }}
            className="space-y-4 mt-4"
          >
            <div>
              <Input
                id="dashboardName"
                value={dashboardName}
                onChange={(e) => setDashboardName(e.target.value)}
                placeholder={translate(`Enter a name for the ${appType === 'workflow' ? 'workflow' : 'dashboard'}`)}
                autoFocus
                required
              />
            </div>
            <DialogFooter>
              <Button
                type="button"
                onClick={() => setShowCreateDialog(false)}
                variant="secondary"
              >
                {translate('Cancel')}
              </Button>
              <Button
                type="submit"
                disabled={creating || !dashboardName.trim()}
                isLoading={creating}
              >
                {translate('Create')}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </MenuProvider>
  )
}
