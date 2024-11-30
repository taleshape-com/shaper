import { z } from 'zod'
import { createFileRoute, Link } from '@tanstack/react-router'
import type { ErrorComponentProps } from '@tanstack/react-router'
import { Dashboard } from '../components/dashboard'
import { Helmet } from 'react-helmet'
import { useNavigate } from '@tanstack/react-router'
import { varsParamSchema } from '../lib/utils'
import { useAuth } from '../lib/auth'

export const Route = createFileRoute('/dashboards/$dashboardId')({
  validateSearch: z.object({
    vars: varsParamSchema,
  }),
  errorComponent: DashboardErrorComponent,
  notFoundComponent: () => {
    return (
      <div>
        <p>Dashboard not found</p>
        <Link to="/" className="underline">
          Go to homepage
        </Link>
      </div>
    )
  },
  component: DashboardViewComponent,
})

function DashboardErrorComponent({ error }: ErrorComponentProps) {
  return (
    <div className="p-4 m-4 bg-red-200 rounded-md">
      <p>{error.message}</p>
    </div>
  )
}

function DashboardViewComponent() {
  const { vars } = Route.useSearch()
  const params = Route.useParams()
  const auth = useAuth()
  const navigate = useNavigate({ from: '/dashboards/$dashboardId' })

  return <>
    <Helmet>
      <title>{params.dashboardId}</title>
      <meta name="description" content={params.dashboardId} />
    </Helmet>
    <Dashboard
      id={params.dashboardId}
      vars={vars}
      getJwt={auth.getJwt}
      onVarsChanged={newVars => {
        navigate({
          search: (old: any) => ({
            ...old,
            vars: newVars,
          }),
        })
      }}
    />
  </>
}
