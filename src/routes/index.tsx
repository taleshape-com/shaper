import { redirect, useRouter } from "@tanstack/react-router";
import { ErrorComponent, createFileRoute, Link } from "@tanstack/react-router";
import type { ErrorComponentProps } from "@tanstack/react-router";
import { Helmet } from "react-helmet";
import { useAuth } from "../lib/auth";
import { IDashboard } from "../lib/dashboard";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeaderCell,
  TableRoot,
  TableRow,
} from "../components/tremor/Table";

type DashboardListResponse = {
  dashboards: IDashboard[];
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

  const handleDelete = async (dashboard: IDashboard) => {
    if (
      !window.confirm(
        `Are you sure you want to delete dashboard "${dashboard.name}"?`,
      )
    ) {
      return;
    }

    try {
      const jwt = await auth.getJwt();
      const response = await fetch(`/api/dashboards/${dashboard.id}`, {
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
    <div className="p-4 max-w-[720px] mx-auto">
      <Helmet>
        <title>Dashboard Overview</title>
        <meta
          name="description"
          content="Show a list of all dashboards"
        />
      </Helmet>
      <div className="mb-4 flex">
        <h2 className="text-2xl font-semibold font-display mb-4 flex-grow">
          Overview
        </h2>
        <Link
          to="/dashboard/new"
          className="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 h-fit"
        >
          New
        </Link>
      </div>
      {data.dashboards.length === 0 ? (
        <p>No dashboards yet</p>
      ) : (
        <TableRoot>
          <Table>
            <TableHead>
              <TableRow>
                <TableHeaderCell>Name</TableHeaderCell>
                <TableHeaderCell className="hidden md:table-cell">
                  Updated
                </TableHeaderCell>
                <TableHeaderCell className="hidden md:table-cell">
                  Actions
                </TableHeaderCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {data.dashboards.map((dashboard) => (
                <TableRow
                  key={dashboard.id}
                  className="group hover:bg-gray-100 dark:hover:bg-gray-800 cursor-pointer transition-colors duration-200"
                  onClick={() =>
                    router.navigate({
                      to: "/dashboards/$dashboardId",
                      params: { dashboardId: dashboard.id },
                    })
                  }
                >
                  <TableCell className="font-medium text-gray-900 dark:text-gray-100">
                    {dashboard.name}
                  </TableCell>
                  <TableCell className="hidden md:table-cell">
                    <div title={new Date(dashboard.updatedAt).toLocaleString()}>
                      {new Date(dashboard.updatedAt).toLocaleDateString()}
                    </div>
                  </TableCell>
                  <TableCell className="hidden md:table-cell">
                    <div className="flex gap-4">
                      <Link
                        to="/dashboards/$dashboardId/edit"
                        params={{ dashboardId: dashboard.id }}
                        className="text-gray-600 hover:text-gray-800 hover:underline"
                        onClick={(e) => e.stopPropagation()}
                      >
                        Edit
                      </Link>
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          handleDelete(dashboard);
                        }}
                        className="text-red-600 hover:text-red-800 hover:underline"
                      >
                        Delete
                      </button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableRoot>
      )}
    </div>
  );
}
