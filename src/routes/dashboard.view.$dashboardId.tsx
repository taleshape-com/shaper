import { z } from "zod";
import { createFileRoute, Link, notFound } from "@tanstack/react-router";
import type { ErrorComponentProps } from "@tanstack/react-router";
import { Result } from "../lib/dashboard";
import DashboardLineChart from "../components/dashboard/DashboardLineChart";
import DashboardTable from "../components/dashboard/DashboardTable";
import { redirect } from "@tanstack/react-router";
import { Helmet } from "react-helmet";
import DashboardBarChart from "../components/dashboard/DashboardBarChart";
import DashboardDropdown from "../components/dashboard/DashboardDropdown";
import { useNavigate } from "@tanstack/react-router";
import DashboardDropdownMulti from "../components/dashboard/DashboardDropdownMulti";

export const Route = createFileRoute("/dashboard/view/$dashboardId")({
  validateSearch: z.object({
    vars: z.record(z.union([z.string(), z.array(z.string())])).optional(),
  }),
  loaderDeps: ({ search: { vars } }) => ({
    vars,
  }),
  loader: async ({ params: { dashboardId }, deps: { vars } }) => {
    const urlVars = Object.entries(vars ?? {}).reduce((acc, [key, value]) => {
      if (Array.isArray(value)) {
        return [...acc, ...value.map((v) => [key, v])];
      }
      return [...acc, [key, value]];
    }, [] as string[][])
    const searchParams = new URLSearchParams(urlVars).toString();
    console.log(searchParams);
    return fetch(`/api/dashboard/${dashboardId}?${searchParams}`)
      .then(async (response) => {
        if (response.status === 401) {
          throw redirect({
            to: "/login",
            search: {
              // Use the current location to power a redirect after login
              // (Do not use `router.state.resolvedLocation` as it can
              // potentially lag behind the actual current location)
              redirect: location.pathname + location.search + location.hash,
            },
          });
        }
        if (response.status === 404) {
          throw notFound();
        }
        if (response.status !== 200) {
          return response
            .json()
            .then(
              (data: {
                Error?: { Type: number; Msg?: string };
                message?: string;
              }) => {
                throw new Error(
                  (
                    data?.Error?.Msg ??
                    data?.Error ??
                    data?.message ??
                    data
                  ).toString(),
                );
              },
            );
        }
        return response.json();
      })
      .then((fetchedData: Result) => {
        return fetchedData;
      });
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
    );
  },
  component: DashboardViewComponent,
});

function DashboardErrorComponent({ error }: ErrorComponentProps) {
  return (
    <div className="p-4 m-4 bg-red-200 rounded-md">
      <p>{error.message}</p>
    </div>
  );
}

function DashboardViewComponent() {
  const { vars } = Route.useSearch();
  const data = Route.useLoaderData();
  const navigate = useNavigate({ from: "/dashboard/view/$dashboardId" });

  if (!data) {
    return <div>Loading...</div>;
  }

  return (
    <div className="w-screen h-screen px-4 py-8 sm:px-6 lg:px-8 overflow-auto">
      <Helmet>
        <title>{data.title}</title>
        <meta name="description" content={data.title} />
      </Helmet>
      <h1 className="mb-8 text-xl text-center">{data.title}</h1>
      <div className="flex flex-col lg:flex-row lg:flex-wrap gap-4">
        {data.queries.length === 0 ? (
          <div>No data to show...</div>
        ) : (
          data.queries.map(({ render, columns, rows }, index) => {
            return (
              <div
                key={index}
                className="lg:w-[calc(50vw-5rem)] h-[calc(50vh-4rem)] lg:h-[calc(100vh-12rem)] mb-24"
              >
                {render.type === "linechart" ? (
                  <>
                    <h2 className="text-lg mb-10 text-center">
                      {render.label}
                    </h2>
                    <DashboardLineChart headers={columns} data={rows} />
                  </>
                ) : render.type === "barchart" ? (
                  <>
                    <h2 className="text-lg mb-10 text-center">
                      {render.label}
                    </h2>
                    <DashboardBarChart headers={columns} data={rows} />
                  </>
                ) : render.type === "dropdown" ? (
                  <>
                    <h2 className="text-lg mb-10 text-center">
                      {render.label}
                    </h2>
                    <DashboardDropdown
                      headers={columns}
                      data={rows}
                      vars={vars}
                      onChange={(newVarz) => {
                        navigate({
                          search: (old: any) => ({
                            ...old,
                            vars: newVarz,
                          }),
                        });
                      }}
                    />
                  </>
                ) : render.type === "dropdownMulti" ? (
                  <DashboardDropdownMulti
                    label={render.label}
                    headers={columns}
                    data={rows}
                    vars={vars}
                    onChange={(newVarz) => {
                      navigate({
                        search: (old: any) => ({
                          ...old,
                          vars: newVarz,
                        }),
                      });
                    }}
                  />
                ) : (
                  <>
                    <h2 className="text-lg mb-10 text-center">
                      {render.label}
                    </h2>
                    <DashboardTable headers={columns} data={rows} />
                  </>
                )}
              </div>
            );
          })
        )}
      </div>
    </div>
  );
}
