import { z } from 'zod'
import { createFileRoute, Link, notFound } from '@tanstack/react-router'
import type { ErrorComponentProps } from '@tanstack/react-router'
import { Result } from '../lib/dashboard'
import { Dashboard } from '../components/dashboard'
import { redirect } from '@tanstack/react-router'
import { Helmet } from 'react-helmet'
import { useNavigate } from '@tanstack/react-router'
import { varsParamSchema, getSearchParamString } from '../lib/utils'

export const Route = createFileRoute('/dashboards/$dashboardId')({
  validateSearch: z.object({
    vars: varsParamSchema,
  }),
  loaderDeps: ({ search: { vars } }) => ({
    vars,
  }),
  loader: async ({ params: { dashboardId }, deps: { vars }, context: { auth: { getJwt } } }) => {
    const jwt = await getJwt();
    const searchParams = getSearchParamString(vars)
    return fetch(`/api/dashboards/${dashboardId}?${searchParams}`, {
      headers: {
        "Content-Type": "application/json",
        Authorization: jwt,
      }
    })
      .then(async (response) => {
        if (response.status === 401) {
          throw redirect({
            to: '/login',
            search: {
              // Use the current location to power a redirect after login
              // (Do not use `router.state.resolvedLocation` as it can
              // potentially lag behind the actual current location)
              redirect: location.pathname + location.search + location.hash,
            },
          })
        }
        if (response.status === 404) {
          throw notFound()
        }
        if (response.status !== 200) {
          return response
            .json()
            .then(
              (data: {
                Error?: { Type: number; Msg?: string }
                message?: string
              }) => {
                throw new Error(
                  (
                    data?.Error?.Msg ??
                    data?.Error ??
                    data?.message ??
                    data
                  ).toString(),
                )
              },
            )
        }
        return response.json()
      })
      .then((fetchedData: Result) => {
        return fetchedData
      })
  },
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
  const data = Route.useLoaderData()
  const navigate = useNavigate({ from: '/dashboards/$dashboardId' })

  return <>
    <Helmet>
      <title>{data.title}</title>
      <meta name="description" content={data.title} />
    </Helmet>
    <Dashboard
      data={data}
      vars={vars}
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
