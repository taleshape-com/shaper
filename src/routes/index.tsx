import { redirect, useNavigate, useRouter } from "@tanstack/react-router";
import { ErrorComponent, createFileRoute, Link } from "@tanstack/react-router";
import type { ErrorComponentProps } from "@tanstack/react-router";
import { Helmet } from "react-helmet";
import { useAuth } from "../lib/auth";

type DashboardListResponse = {
  dashboards: string[];
};

export const Route = createFileRoute("/")({
  loader: async ({
    context: {
      auth: { getJwt },
    },
  }) => {
    const jwt = await getJwt();
    return fetch(`/api/dashboards`, {
      headers: {
        "Content-Type": "application/json",
        Authorization: jwt,
      },
    })
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
  const auth = useAuth();
  const router = useRouter();

  const handleDelete = async (dashboardId: string) => {
    if (
      !window.confirm(
        `Are you sure you want to delete dashboard "${dashboardId}"?`,
      )
    ) {
      return;
    }

    try {
      const jwt = await auth.getJwt();
      const response = await fetch(`/api/dashboards/${dashboardId}`, {
        method: "DELETE",
        headers: {
          Authorization: jwt,
        },
      });

      if (!response.ok) {
        throw new Error("Failed to delete dashboard");
      }

      // Reload the page to refresh the list
      router.invalidate();
    } catch (err) {
      alert(
        "Error deleting dashboard: " +
        (err instanceof Error ? err.message : "Unknown error"),
      );
    }
  };

  if (!data) {
    return <div className="p-2">Loading dashboards...</div>;
  }

  return (
    <div className="p-4">
      <Helmet>
        <title>Dashboard Overview</title>
        <meta
          name="description"
          content="Show a list of all available dashboards"
        />
      </Helmet>
      <h2 className="text-2xl font-bold mb-4">Available Dashboards</h2>
      <Link
        to="/dashboard/new"
        className="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600"
      >
        New Dashboard
      </Link>
      {data.dashboards.length === 0 ? (
        <p>No dashboards available.</p>
      ) : (
        <ul className="space-y-2">
          {data.dashboards.map((dashboard) => (
            <li
              key={dashboard}
              className="bg-gray-100 p-2 rounded flex justify-between items-center hover:bg-gray-300"
            >
              <Link
                to="/dashboards/$dashboardId"
                params={{ dashboardId: dashboard }}
                className="text-blue-600 hover:underline"
              >
                {dashboard}
              </Link>
              <div className="space-x-2">
                <Link
                  to="/dashboards/$dashboardId/edit"
                  params={{ dashboardId: dashboard }}
                  className="text-gray-600 hover:text-gray-800"
                >
                  Edit
                </Link>
                <button
                  onClick={() => handleDelete(dashboard)}
                  className="text-red-600 hover:text-red-800"
                >
                  Delete
                </button>
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
