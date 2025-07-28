import { Editor } from '@monaco-editor/react'
import * as monaco from 'monaco-editor'
import { z } from 'zod'
import {
  createFileRoute,
  isRedirect,
  useNavigate,
} from '@tanstack/react-router'
import { useCallback, useEffect, useRef, useState, useContext } from 'react'
import { Helmet } from 'react-helmet'
import { useAuth } from '../lib/auth'
import { Dashboard } from '../components/dashboard'
import { useDebouncedCallback } from 'use-debounce'
import {
  cx,
  focusRing,
  getSearchParamString,
  hasErrorInput,
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
import { DarkModeContext } from '../contexts/DarkModeContext'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '../components/tremor/Dialog'
import { Input } from '../components/tremor/Input'
import "../lib/editorInit";

const defaultQuery = '-- Enter your SQL query here'

export const Route = createFileRoute('/new')({
  validateSearch: z.object({
    vars: varsParamSchema,
  }),
  component: NewDashboard,
})

function NewDashboard() {
  const { vars } = Route.useSearch()
  const auth = useAuth()
  const queryApi = useQueryApi()
  const navigate = useNavigate({ from: '/new' })
  const [editorQuery, setEditorQuery] = useState(defaultQuery)
  const [runningQuery, setRunningQuery] = useState(defaultQuery)
  const [creating, setCreating] = useState(false)
  const [hasVariableError, setHasVariableError] = useState(false)
  const [previewData, setPreviewData] = useState<Result | undefined>(undefined)
  const [previewError, setPreviewError] = useState<string | null>(null)
  const [isPreviewLoading, setIsPreviewLoading] = useState(false)
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [dashboardName, setDashboardName] = useState('')
  const { toast } = useToast()
  const { isDarkMode } = useContext(DarkModeContext)

  // Check for unsaved changes when component mounts
  useEffect(() => {
    const unsavedContent = editorStorage.getChanges('new')
    if (unsavedContent) {
      setEditorQuery(unsavedContent)
      setRunningQuery(unsavedContent)
    }
  }, [])

  const previewDashboard = useCallback(async () => {
    setPreviewError(null)
    setIsPreviewLoading(true)
    try {
      const searchParams = getSearchParamString(vars)
      const data = await queryApi(`query/dashboard?${searchParams}`, {
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

  useEffect(() => {
    previewDashboard()
  }, [previewDashboard])

  const handleRun = useCallback(() => {
    if (isPreviewLoading) {
      return;
    }
    if (editorQuery !== runningQuery) {
      setRunningQuery(editorQuery)
    } else {
      previewDashboard()
    }
  }, [editorQuery, runningQuery, previewDashboard, isPreviewLoading])

  const handleRunRef = useRef(handleRun)

  useEffect(() => {
    handleRunRef.current = handleRun
  }, [handleRun])

  // We handle this command in monac and outside
  // so even if the editor is not focused the shortcut works
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Check for Ctrl+Enter or Cmd+Enter (Mac)
      if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
        e.preventDefault()
        handleRun()
      }
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [handleRun])

  const handleQueryChange = (value: string | undefined) => {
    const newQuery = value || ''
    // Save to localStorage
    if (newQuery !== defaultQuery && newQuery.trim() !== '') {
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

  const onVariablesEdit = useDebouncedCallback((value) => {
    auth.updateVariables(value).then(
      (ok) => {
        setHasVariableError(!ok)
        if (ok) {
          // Refresh preview when variables change
          previewDashboard()
        }
      },
      () => {
        setHasVariableError(true)
      },
    )
  }, 500)

  return (
    <MenuProvider isNewPage>
      <Helmet>
        <title>New Dashboard</title>
      </Helmet>

      <div className="h-dvh flex flex-col">
        <div className="h-[42dvh] flex flex-col overflow-y-hidden max-h-[90dvh] min-h-[12dvh] resize-y shrink-0 shadow-sm dark:shadow-none">
          <div className="flex items-center p-2 border-b border-cb dark:border-none">
            <MenuTrigger className="pr-2">
              <div className="mt-6 px-4 w-full">
                <label>
                  <span className="text-lg font-medium font-display ml-1 mb-2 block">
                    {translate('Variables')}
                  </span>
                  <textarea
                    className={cx(
                      'w-full px-3 py-1.5 bg-cbg dark:bg-dbg text-sm border border-cb dark:border-db shadow-sm outline-none ring-0 rounded-md font-mono resize-none h-28',
                      focusRing,
                      hasVariableError && hasErrorInput,
                    )}
                    onChange={(event) => {
                      onVariablesEdit(event.target.value)
                    }}
                    defaultValue={JSON.stringify(auth.variables, null, 2)}
                  ></textarea>
                </label>
              </div>
            </MenuTrigger>

            <h1 className="text-xl font-semibold font-display flex-grow">
              {translate('New Dashboard')}
            </h1>

            <div className="space-x-2">
              <Tooltip showArrow={false} asChild content="Create Dashboard">
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
            <Editor
              height="100%"
              defaultLanguage="sql"
              value={editorQuery}
              onChange={handleQueryChange}
              theme={isDarkMode ? 'vs-dark' : 'light'}
              options={{
                minimap: { enabled: false },
                fontSize: 14,
                lineNumbers: 'on',
                scrollBeyondLastLine: false,
                wordWrap: 'on',
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
                    handleRunRef.current()
                  },
                )
              }}
            />
          </div>
        </div>

        <div className="flex-grow overflow-scroll relative pt-1">
          {previewError && (
            <div className="fixed w-full h-full p-4 z-50 backdrop-blur-sm flex justify-center">
              <div className="p-4 bg-red-100 text-red-700 rounded mt-32 h-fit">
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

      <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{translate('Create Dashboard')}</DialogTitle>
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
                placeholder={translate('Enter a name for the dashboard')}
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
