import z from "zod";
import { useState } from "react";
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
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../components/tremor/Dialog";
import { Button } from "../components/tremor/Button";
import { Label } from "../components/tremor/Label";
import { Input } from "../components/tremor/Input";
import { useToast } from "../hooks/useToast";
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

const getInviteLink = (code: string) => {
  const baseUrl = window.location.origin;
  return `${baseUrl}/signup?code=${code}`;
};

function UsersManagement() {
  const router = useRouter();
  const queryApi = useQueryApi();
  const data = Route.useLoaderData();
  const { sort, order } = Route.useSearch();
  const navigate = useNavigate({ from: "/admin/users" });
  const [showInviteDialog, setShowInviteDialog] = useState(false);
  const [inviteCode, setInviteCode] = useState<{ code: string; email: string } | null>(null);
  const { toast } = useToast();


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

  const handleCreateInvite = async (email: string) => {
    try {
      const data = await queryApi("/api/invites", {
        method: "POST",
        body: { email },
      });
      setInviteCode({ code: data.code, email });
    } catch (error) {
      if (isRedirect(error)) {
        return navigate(error);
      }
      toast({
        title: translate("Error"),
        description:
          error instanceof Error
            ? error.message
            : translate("An error occurred"),
        variant: "error",
      });
    }
  };

  return (
    <>
      <div className="flex justify-between items-center mb-4">
        <h2 className="text-xl font-semibold">
          {translate("User Management")}
        </h2>
        <Button onClick={() => setShowInviteDialog(true)}>
          {translate("Invite User")}
        </Button>
      </div>

      {showInviteDialog && (
        <Dialog
          open={true}
          onOpenChange={() => {
            setShowInviteDialog(false);
            setInviteCode(null);
          }}
        >
          <DialogContent className="sm:max-w-lg">
            <DialogHeader>
              <DialogTitle>
                {inviteCode
                  ? translate("Invite Link")
                  : translate("Invite User")}
              </DialogTitle>
              <DialogDescription>
                {inviteCode
                  ? translate(
                    "Share this link with %%",
                  ).replace("%%", inviteCode.email)
                  : translate(
                    "Enter the email address of the user you want to invite",
                  )}
              </DialogDescription>
            </DialogHeader>

            {inviteCode ? (
              <div>
                <div className="flex items-center gap-2 my-4">
                  <code className="bg-gray-100 dark:bg-gray-700 p-2 rounded flex-grow overflow-hidden text-ellipsis">
                    {getInviteLink(inviteCode.code)}
                  </code>
                  <Button
                    onClick={() => {
                      navigator.clipboard.writeText(getInviteLink(inviteCode.code));
                      toast({
                        title: translate("Success"),
                        description: translate(
                          "Invite link copied to clipboard",
                        ),
                      });
                    }}
                    variant="secondary"
                  >
                    {translate("Copy")}
                  </Button>
                </div>
                <DialogFooter>
                  <Button
                    onClick={() => {
                      setShowInviteDialog(false);
                      setInviteCode(null);
                    }}
                  >
                    {translate("Close")}
                  </Button>
                </DialogFooter>
              </div>
            ) : (
              <form
                className="space-y-4"
                onSubmit={(e) => {
                  e.preventDefault();
                  const formData = new FormData(e.currentTarget);
                  handleCreateInvite(formData.get("email") as string);
                }}
              >
                <div className="space-y-2">
                  <Label htmlFor="email">{translate("Email")}</Label>
                  <Input
                    id="email"
                    name="email"
                    type="email"
                    required
                    placeholder="user@example.com"
                  />
                </div>

                <DialogFooter className="mt-6">
                  <DialogClose asChild>
                    <Button type="button" variant="secondary">
                      {translate("Cancel")}
                    </Button>
                  </DialogClose>
                  <Button type="submit">{translate("Create Invite")}</Button>
                </DialogFooter>
              </form>
            )}
          </DialogContent>
        </Dialog>
      )}

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
