import { ErrorComponent, createFileRoute, Link } from "@tanstack/react-router";
import type { ErrorComponentProps } from "@tanstack/react-router";
import { Result } from "../lib/dashboard";
import DashboardLineChart from "../components/dashboard/DashboardLineChart";
import DashboardTable from "../components/dashboard/DashboardTable";

export const Route = createFileRoute("/dashboard/view/$dashboardId")({
  loader: async ({ params: { dashboardId } }) => {
    return fetch(
      `${import.meta.env.VITE_API_URL || ""}/api/dashboard/${dashboardId}`,
    )
      .then((response) => {
        if (response.status !== 200) {
          return response
            .json()
            .then((data: { Error: { Type: number; Msg: string } }) => {
              throw new Error(data.Error.Msg);
            });
        }
        return response.json();
      })
      .then((fetchedData: Result) => {
        return fetchedData;
      });
  },
  errorComponent: DashboardErrorComponent as any,
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
  return <ErrorComponent error={error} />;
}

function DashboardViewComponent() {
  const data = Route.useLoaderData();

  let nextTitle: string | undefined = undefined;

  if (!data) {
    return <div>Loading...</div>;
  }

  return (
    <div className="w-screen h-screen px-4 py-8 sm:px-6 lg:px-8 overflow-auto">
      <h1 className="mb-8 text-xl text-center">{data.title}</h1>
      <div className="flex flex-col lg:flex-row lg:flex-wrap gap-4">
        {data.queries.length === 0 ? (
          <div>No data to show...</div>
        ) : (
          data.queries.map(({ render, columns, rows }, index) => {
            if (render.type === "title") {
              nextTitle = rows[0][0] as string;
              return;
            }
            let title: string | undefined = undefined;
            if (nextTitle) {
              title = nextTitle;
              nextTitle = undefined;
            }
            return (
              <div
                key={index}
                className="lg:w-[calc(50vw-5rem)] h-[calc(50vh-4rem)] lg:h-[calc(100vh-12rem)]"
              >
                <h2 className="text-lg mb-10 text-center">{title}</h2>
                {render.type === "line" ? (
                  <DashboardLineChart
                    headers={columns}
                    data={rows}
                    xaxis={render.xAxis}
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
