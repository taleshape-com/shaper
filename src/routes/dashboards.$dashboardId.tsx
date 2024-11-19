import { z } from 'zod'
import { createFileRoute, Link, notFound } from '@tanstack/react-router'
import type { ErrorComponentProps } from '@tanstack/react-router'
import { Result } from '../lib/dashboard'
import DashboardLineChart from '../components/dashboard/DashboardLineChart'
import DashboardTable from '../components/dashboard/DashboardTable'
import { redirect } from '@tanstack/react-router'
import { Helmet } from 'react-helmet'
import DashboardBarChart from '../components/dashboard/DashboardBarChart'
import DashboardDropdown from '../components/dashboard/DashboardDropdown'
import { useNavigate } from '@tanstack/react-router'
import DashboardDropdownMulti from '../components/dashboard/DashboardDropdownMulti'
import { cx } from '../lib/utils'
import DashboardButton from '../components/dashboard/DashboardButton'
import DashboardDatePicker from '../components/dashboard/DashboardDatePicker'
import DashboardDateRangePicker from '../components/dashboard/DashboardDateRangePicker'
import DashboardValue from '../components/dashboard/DashboardValue'
import { Card } from "../components/tremor/Card";
import { translate } from '../lib/translate'

const zVars = z.record(z.union([z.string(), z.array(z.string())])).optional()

export const Route = createFileRoute('/dashboards/$dashboardId')({
  validateSearch: z.object({
    vars: zVars,
  }),
  loaderDeps: ({ search: { vars } }) => ({
    vars,
  }),
  loader: async ({ params: { dashboardId }, deps: { vars } }) => {
    const searchParams = getSearchParamString(vars)
    return fetch(`/api/dashboards/${dashboardId}?${searchParams}`)
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

const getSearchParamString = (vars: typeof zVars['_type']) => {
  const urlVars = Object.entries(vars ?? {}).reduce((acc, [key, value]) => {
    if (Array.isArray(value)) {
      return [...acc, ...value.map((v) => [key, v])]
    }
    return [...acc, [key, value]]
  }, [] as string[][])
  return new URLSearchParams(urlVars).toString()
}

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

  if (!data) {
    return <div>Loading...</div>
  }
  const onDropdownChange = (newVars: Record<string, string | string[]>) => {
    navigate({
      search: (old: any) => ({
        ...old,
        vars: newVars,
      }),
    })
  }

  const searchParams = getSearchParamString(vars)

  const sections: Result['sections'] =
    data.sections.length === 0
      ? [
        {
          type: 'header',
          queries: [],
        },
      ]
      : data.sections[0].type !== 'header'
        ? [
          {
            type: 'header',
            queries: [],
          },
          ...data.sections,
        ]
        : data.sections
  const numContentSections = sections.filter(section => section.type === 'content').length
  return (
    <div className="mb-16 mt-1">
      <Helmet>
        <title>{data.title}</title>
        <meta name="description" content={data.title} />
      </Helmet>
      {sections.map((section, index) => {
        if (section.type === 'header') {
          const queries = section.queries.filter(query => query.rows.length > 0)
          return (
            <section
              key={index}
              className={cx('flex flex-wrap items-center pr-2 mx-1', {
                'mt-1 border-t': index !== 0 && section.title,
                'py-1 mb-2': section.queries.length > 0 || section.title,
              })}
            >
              {index === 0 ? (
                <h1 className="text-2xl text-slate-900 flex-grow py-1 ml-2 mr-4 w-full sm:w-fit">
                  {data.title}
                </h1>
              ) : null}
              {section.title ? (
                <h1 className="text-lg text-slate-900 flex-grow text-left py-1 ml-2 mr-4 mt-5 w-full sm:w-fit">
                  {section.title}
                </h1>
              ) : (
                <div className="sm:flex-grow"></div>
              )}
              {queries.map(({ render, columns, rows }, index) => {
                if (render.type === 'dropdown') {
                  return (
                    <DashboardDropdown
                      key={index}
                      label={render.label}
                      headers={columns}
                      data={rows}
                      vars={vars}
                      onChange={onDropdownChange}
                    />
                  )
                }
                if (render.type === 'dropdownMulti') {
                  return (
                    <DashboardDropdownMulti
                      key={index}
                      label={render.label}
                      headers={columns}
                      data={rows}
                      vars={vars}
                      onChange={onDropdownChange}
                    />
                  )
                }
                if (render.type === 'button') {
                  return (
                    <DashboardButton
                      key={index}
                      label={render.label}
                      headers={columns}
                      data={rows}
                      searchParams={searchParams}
                    />
                  )
                }
                if (render.type === 'datepicker') {
                  return (
                    <DashboardDatePicker
                      key={index}
                      label={render.label}
                      headers={columns}
                      data={rows}
                      vars={vars}
                      onChange={onDropdownChange}
                    />
                  )
                }
                if (render.type === 'daterangePicker') {
                  return (
                    <DashboardDateRangePicker
                      key={index}
                      label={render.label}
                      headers={columns}
                      data={rows}
                      vars={vars}
                      onChange={onDropdownChange}
                    />
                  )
                }
              })}
            </section>
          )
        }
        const numQueriesInSection = section.queries.length
        return (
          <section
            key={index}
            className={cx(
              'grid grid-cols-1 ml-3', {
              'sm:grid-cols-2': numQueriesInSection > 1,
              'lg:grid-cols-2': numQueriesInSection === 2 || (numContentSections === 1 && numQueriesInSection === 4),
              'lg:grid-cols-3': numQueriesInSection > 4 || numQueriesInSection === 3 || (numQueriesInSection === 4 && numContentSections > 1),
              'xl:grid-cols-4': (numQueriesInSection === 4 && numContentSections > 1) || numQueriesInSection === 7 || numQueriesInSection === 8 || numQueriesInSection > 9
            })}
          >
            {section.queries.map(({ render, columns, rows }, index) => {
              return (
                <Card key={index} className={cx(
                  "mr-3 mb-3 p-3 h-[calc(50vh-2.6rem)] min-h-[18rem]",
                  {
                    'h-[calc(65vh-4.7rem)] sm:h-[calc(100vh-4.7rem)]': numQueriesInSection === 1,
                    'lg:h-[calc(100vh-4.7rem)]': numContentSections === 1 && numQueriesInSection === 2,
                  })}>
                  {render.label ? <h2 className="text-md mb-2 text-center text-slate-900">
                    {render.label}
                  </h2>
                    : null
                  }
                  <div className={cx({
                    'h-[calc(100%-2rem)]': render.label,
                    'h-full': !render.label,
                  })}>
                    {
                      rows.length === 0 ? (
                        <div className="h-full py-1 px-3 flex items-center justify-center text-slate-600">
                          {translate('No data available')}
                        </div>
                      ) :
                        render.type === 'linechart' ?
                          <DashboardLineChart
                            key={index}
                            headers={columns}
                            data={rows}
                          /> :
                          render.type === 'barchartHorizontal' || render.type === 'barchartHorizontalStacked' || render.type === 'barchartVertical' || render.type === 'barchartVerticalStacked' ? (
                            <DashboardBarChart
                              stacked={render.type === 'barchartHorizontalStacked' || render.type === 'barchartVerticalStacked'}
                              vertical={render.type === 'barchartVertical' || render.type === 'barchartVerticalStacked'}
                              key={index}
                              headers={columns}
                              data={rows}
                            />
                          )
                            : render.type === 'value' ? (
                              <DashboardValue
                                key={index}
                                headers={columns}
                                data={rows}
                              />
                            ) :
                              (
                                <DashboardTable
                                  key={index}
                                  headers={columns}
                                  data={rows}
                                />
                              )}
                  </div>
                </Card>
              )
            })}
          </section>
        )
      })}
      {numContentSections === 0 ? (
        <div className="text-center text-slate-600 leading-[calc(70vh)]">
          Nothing to show yet ...
        </div>
      ) : null}
    </div>
  )
}
