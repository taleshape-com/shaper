import z from "zod";
import {
  createFileRoute,
  isRedirect,
  useNavigate,
} from "@tanstack/react-router";
import { translate } from "../lib/translate";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeaderCell,
  TableRoot,
  TableRow,
} from "../components/tremor/Table";
import { RiSortAsc, RiSortDesc } from "@remixicon/react";
import { useRouter } from "@tanstack/react-router";
import { useQueryApi } from "../hooks/useQueryApi";

interface IUser {
  id: string;
  email: string;
  name: string;
  createdAt: string;
}

type UserListResponse = {
  users: IUser[];
};

export const Route = createFileRoute("/admin/users")({
  validateSearch: z.object({
    sort: z.enum(["name", "email", "created"]).optional(),
    order: z.enum(["asc", "desc"]).optional(),
  }),
  loaderDeps: ({ search: { sort, order } }) => ({
    sort,
    order,
  }),
  loader: async ({
    context: { queryApi },
    deps: { sort = "created", order = "desc" },
  }) => {
    return queryApi(
      `/api/users?sort=${sort}&order=${order}`,
    ) as Promise<UserListResponse>;
  },
  component: UsersManagement,
});

function UsersManagement() {
  const router = useRouter();
  const queryApi = useQueryApi();
  const data = Route.useLoaderData();
  const { sort, order } = Route.useSearch();
  const navigate = useNavigate({ from: "/admin/users" });

  const handleSort = (field: "name" | "email" | "created") => {
    const newOrder =
      field === (sort ?? "created")
        ? (order ?? "desc") === "asc"
          ? "desc"
          : "asc"
        : field === "created"
          ? "desc"
          : "asc";

    navigate({
      replace: true,
      search: (prev) => ({
        ...prev,
        sort: field === "created" ? undefined : field,
        order:
          field === "created" && newOrder === "desc" ? undefined : newOrder,
      }),
    });
  };

  const SortIcon = ({ field }: { field: "name" | "email" | "created" }) => {
    if (field !== (sort ?? "created")) return null;
    return (order ?? "desc") === "asc" ? (
      <RiSortAsc className="inline size-4" />
    ) : (
      <RiSortDesc className="inline size-4" />
    );
  };

  const handleDelete = async (user: IUser) => {
    if (
      !window.confirm(
        translate("Are you sure you want to delete the user %%?").replace(
          "%%",
          user.email,
        ),
      )
    ) {
      return;
    }

    try {
      await queryApi(`/api/users/${user.id}`, {
        method: "DELETE",
      });
      // Reload the page to refresh the list
      router.invalidate();
    } catch (err) {
      if (isRedirect(err)) {
        return navigate(err);
      }
      alert(
        "Error deleting user: " +
          (err instanceof Error ? err.message : "Unknown error"),
      );
    }
  };

  if (!data) {
    return <div className="p-2">{translate("Loading users...")}</div>;
  }

  return (
    <>
      <h2 className="text-xl font-semibold mb-4">
        {translate("User Management")}
      </h2>

      {data.users.length === 0 ? (
        <p className="text-gray-500">{translate("No users found")}</p>
      ) : (
        <TableRoot>
          <Table>
            <TableHead>
              <TableRow>
                <TableHeaderCell
                  onClick={() => handleSort("name")}
                  className="text-md text-ctext dark:text-dtext cursor-pointer hover:bg-cbga dark:hover:bg-dbga"
                >
                  {translate("Name")} <SortIcon field="name" />
                </TableHeaderCell>
                <TableHeaderCell
                  onClick={() => handleSort("email")}
                  className="text-md text-ctext dark:text-dtext cursor-pointer hover:bg-cbga dark:hover:bg-dbga"
                >
                  {translate("Email")} <SortIcon field="email" />
                </TableHeaderCell>
                <TableHeaderCell
                  onClick={() => handleSort("created")}
                  className="text-md text-ctext dark:text-dtext hidden md:table-cell cursor-pointer hover:bg-cbga dark:hover:bg-dbga"
                >
                  {translate("Created")} <SortIcon field="created" />
                </TableHeaderCell>
                <TableHeaderCell className="text-md text-ctext dark:text-dtext">
                  {translate("Actions")}
                </TableHeaderCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {data.users.map((user) => (
                <TableRow key={user.id}>
                  <TableCell className="font-medium text-ctext dark:text-dtext">
                    {user.name}
                  </TableCell>
                  <TableCell className="text-ctext2 dark:text-dtext2">
                    {user.email}
                  </TableCell>
                  <TableCell className="hidden md:table-cell text-ctext2 dark:text-dtext2">
                    <div title={new Date(user.createdAt).toLocaleString()}>
                      {new Date(user.createdAt).toLocaleDateString()}
                    </div>
                  </TableCell>
                  <TableCell>
                    <button
                      onClick={() => handleDelete(user)}
                      className="text-cerr dark:text-derr opacity-90 hover:opacity-100 hover:underline"
                    >
                      {translate("Delete")}
                    </button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableRoot>
      )}
    </>
  );
}
