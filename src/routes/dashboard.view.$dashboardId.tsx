import { createFileRoute, Link, notFound } from "@tanstack/react-router";
import type { ErrorComponentProps } from "@tanstack/react-router";
import { Result } from "../lib/dashboard";
import DashboardLineChart from "../components/dashboard/DashboardLineChart";
import DashboardTable from "../components/dashboard/DashboardTable";
import { redirect } from "@tanstack/react-router";
import { Helmet } from "react-helmet";
import DashboardBarChart from "../components/dashboard/DashboardBarChart";

export const Route = createFileRoute("/dashboard/view/$dashboardId")({
  loader: async ({ params: { dashboardId } }) => {
    return fetch(`/api/dashboard/${dashboardId}`)
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
  const data = Route.useLoaderData();

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
                className="lg:w-[calc(50vw-5rem)] h-[calc(50vh-4rem)] lg:h-[calc(50vh-10rem)] mb-24"
              >
                <h2 className="text-lg mb-10 text-center">{render.label}</h2>
                {render.type === "linechart" ? (
                  <DashboardLineChart
                    headers={columns}
                    data={rows}
                    xaxis={render.xAxis}
                    yaxis={render.yAxis}
                    categoryIndex={render.categoryIndex}
                  />
                ) : render.type === "barchart" ? (
                  <DashboardBarChart
                    headers={columns}
                    data={rows}
                    xaxis={render.xAxis}
                    yaxis={render.yAxis}
                    categoryIndex={render.categoryIndex}
                  />
                ) : (
                  <DashboardTable headers={columns} data={rows} />
                )}
              </div>
            );
          })
        )}
      </div>
    </div>
  );
}
