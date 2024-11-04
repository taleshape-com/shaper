import { redirect } from "@tanstack/react-router";
import { ErrorComponent, createFileRoute, Link } from "@tanstack/react-router";
import type { ErrorComponentProps } from "@tanstack/react-router";

type DashboardListResponse = {
  dashboards: string[];
};

export const Route = createFileRoute("/")({
  loader: async () => {
    return fetch(`/api/dashboards`)
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
        if (response.status !== 200) {
          return response
            .json()
            .then((data: { Error: { Type: number; Msg: string } }) => {
              throw new Error(data.Error.Msg);
            });
        }
        return response.json();
      })
      .then((fetchedData: DashboardListResponse) => {
        return fetchedData;
      });
  },
  errorComponent: DashboardErrorComponent as any,
  component: Index,
});

function DashboardErrorComponent({ error }: ErrorComponentProps) {
  return <ErrorComponent error={error} />;
}

function Index() {
  const data = Route.useLoaderData();

  if (!data) {
    return <div className="p-2">Loading dashboards...</div>;
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
  );
}
