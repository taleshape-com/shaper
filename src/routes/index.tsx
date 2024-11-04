import { ErrorComponent, createFileRoute, Link } from '@tanstack/react-router'
import type { ErrorComponentProps } from '@tanstack/react-router'

type DashboardListResponse = {
  dashboards: string[]
}

export const Route = createFileRoute('/')({
  loader: async () => {
    return fetch(`${import.meta.env.VITE_API_URL || ''}/api/dashboards`)
      .then((response) => response.json())
      .then((fetchedData: DashboardListResponse) => {
        return fetchedData
      })
  },
  errorComponent: DashboardErrorComponent as any,
  component: Index,
})

function DashboardErrorComponent({ error }: ErrorComponentProps) {
  return <ErrorComponent error={error} />
}

function Index() {
  const data = Route.useLoaderData()

  if (!data) {
    return <div className="p-2">Loading dashboards...</div>
  }

  return (
    <div className="p-4">
      <h2 className="text-2xl font-bold mb-4">Available Dashboards</h2>
      {data.dashboards.length === 0 ? (
        <p>No dashboards available.</p>
      ) : (
        <ul className="space-y-2">
          {data.dashboards.map((dashboard) => (
            <li key={dashboard} className="bg-gray-100 p-2 rounded">
              <Link
                to="/dashboard/view/$dashboardId"
                params={{ dashboardId: dashboard }}
                className="text-blue-600 hover:underline"
              >
                {dashboard}
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
