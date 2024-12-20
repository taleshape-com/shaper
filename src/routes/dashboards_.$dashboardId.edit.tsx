import { createFileRoute, Link } from '@tanstack/react-router'
import { useCallback, useState } from 'react'
import { Helmet } from 'react-helmet'
import { useAuth } from '../lib/auth'

export const Route = createFileRoute('/dashboards_/$dashboardId/edit')({
  loader: async ({
    params: { dashboardId },
    context: {
      auth: { getJwt },
    },
  }) => {
    const jwt = await getJwt()
    const response = await fetch(`/api/dashboards/${dashboardId}/query`, {
      headers: {
        Authorization: jwt,
      },
    })
    if (!response.ok) {
      throw new Error('Failed to load dashboard query')
    }
    const data = await response.json()
    return data.content
  },
  component: DashboardEditor,
})

function DashboardEditor() {
  const params = Route.useParams()
  const content = Route.useLoaderData()
  const auth = useAuth()
  const [query, setQuery] = useState(content)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleSave = useCallback(async () => {
    setSaving(true)
    setError(null)
    try {
      const jwt = await auth.getJwt()
      const response = await fetch(
        `/api/dashboards/${params.dashboardId}/query`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            Authorization: jwt,
          },
          body: JSON.stringify({ content: query }),
        },
      )

      if (!response.ok) {
        throw new Error('Failed to save dashboard query')
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setSaving(false)
    }
  }, [auth, params.dashboardId, query])

  return (
    <div className="p-4">
      <Helmet>
        <title>Edit {params.dashboardId}</title>
      </Helmet>

      <div className="flex justify-between items-center mb-4">
        <h1 className="text-2xl font-bold">
          Edit Dashboard: {params.dashboardId}
        </h1>
        <div className="space-x-2">
          <Link
            to="/dashboards/$dashboardId"
            params={{ dashboardId: params.dashboardId }}
            className="px-4 py-2 text-gray-600 hover:text-gray-800"
          >
            View
          </Link>
          <button
            onClick={handleSave}
            disabled={saving}
            className="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 disabled:opacity-50"
          >
            {saving ? 'Saving...' : 'Save'}
          </button>
        </div>
      </div>

      {error && (
        <div className="mb-4 p-4 bg-red-100 text-red-700 rounded">{error}</div>
      )}

      <textarea
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        className="w-full h-[calc(100vh-200px)] p-4 font-mono text-sm border rounded"
        spellCheck={false}
      />
    </div>
  )
}
