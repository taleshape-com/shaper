// SPDX-License-Identifier: MPL-2.0

import z from "zod";
import {
  createFileRoute,
  isRedirect,
  useNavigate,
} from "@tanstack/react-router";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeaderCell,
  TableRoot,
  TableRow,
} from "../components/tremor/Table";
import { RiGroupLine, RiSortAsc, RiSortDesc, RiUserAddFill, RiUserAddLine } from "@remixicon/react";
import { useState } from "react";
import { Button } from "../components/tremor/Button";
import { useToast } from "../hooks/useToast";
import { useRouter } from "@tanstack/react-router";
import { useQueryApi } from "../hooks/useQueryApi";
import { Callout } from "../components/tremor/Callout";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "../components/tremor/Dialog";
import { Input } from "../components/tremor/Input";
import { Label } from "../components/tremor/Label";
import { Tooltip } from "../components/tremor/Tooltip";
import { getSystemConfig, fetchSystemConfig } from "../lib/system";

interface IUser {
  id: string;
  email: string;
  name: string;
  createdAt: string;
}

interface IInvite {
  code: string;
  email: string;
  createdAt: string;
}

type UserListResponse = {
  users: IUser[];
  invites: IInvite[];
  inviteValidTimeInSeconds: number;
};

interface InviteState {
  isExpired: boolean
  expiresIn?: string
}

function getInviteState(createdAt: string, validTimeInSeconds: number): InviteState {
  const createdTime = new Date(createdAt).getTime()
  const expirationTime = createdTime + (validTimeInSeconds * 1000)
  const now = Date.now()
  const isExpired = now > expirationTime

  if (isExpired) {
    return { isExpired: true }
  }

  const timeLeft = expirationTime - now
  const days = Math.floor(timeLeft / (1000 * 60 * 60 * 24))
  const hours = Math.floor((timeLeft % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60))

  let expiresIn = ''
  if (days > 0) {
    expiresIn = ('Expires in %% days').replace('%%', days.toString())
  } else if (hours > 0) {
    expiresIn = ('Expires in %% hours').replace('%%', hours.toString())
  } else {
    expiresIn = 'Expires soon'
  }

  return { isExpired: false, expiresIn }
}

export const Route = createFileRoute("/admin/")({
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
      `users?sort=${sort}&order=${order}`,
    ) as Promise<UserListResponse>;
  },
  component: UsersManagement,
});

const getInviteLink = (code: string) => {
  const baseUrl = window.location.origin;
  const basePath = window.shaper.defaultBaseUrl;
  return `${baseUrl}${basePath}signup?code=${code}`;
};

function UsersManagement() {
  const router = useRouter();
  const data = Route.useLoaderData();
  const { sort, order } = Route.useSearch();
  const navigate = useNavigate({ from: "/admin" });
  const [showInviteDialog, setShowInviteDialog] = useState(false);
  const [inviteCode, setInviteCode] = useState<{
    code: string;
    email: string;
  } | null>(null);
  const [deleteUserDialog, setDeleteUserDialog] = useState<IUser | null>(null);
  const [deleteInviteDialog, setDeleteInviteDialog] = useState<IInvite | null>(null);
  const { toast } = useToast();
  const queryApi = useQueryApi();

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
    try {
      await queryApi(`users/${user.id}`, {
        method: "DELETE",
      });
      // Reload the page to refresh the list
      router.invalidate();
      toast({
        title: "Success",
        description: "User deleted successfully",
      });
    } catch (err) {
      if (isRedirect(err)) {
        return navigate(err.options);
      }
      toast({
        title: "Error",
        description: err instanceof Error ? err.message : "Unknown error",
        variant: "error",
      });
    }
  };

  if (!data) {
    return <div className="p-2">Loading users...</div>;
  }

  const handleCreateInvite = async (email: string) => {
    try {
      const data = await queryApi("invites", {
        method: "POST",
        body: { email },
      });
      setInviteCode({ code: data.code, email });
      router.invalidate();
    } catch (error) {
      if (isRedirect(error)) {
        return navigate(error.options);
      }
      toast({
        title: "Error",
        description:
          error instanceof Error
            ? error.message
            : "An error occurred",
        variant: "error",
      });
    }
  };

  return (
    <>
      <div className="flex justify-between items-center mb-4">
        <h2 className="text-xl font-semibold">
          <RiGroupLine className="size-4 inline mr-1 -mt-1" />
          User Management
        </h2>
        {getSystemConfig().loginRequired && (
          <Button onClick={() => setShowInviteDialog(true)}>
            <RiUserAddLine className="size-4 inline mr-1 -ml-0.5 -mt-0.5" />
            Invite User
          </Button>
        )}
      </div>

      {!getSystemConfig().loginRequired && (
        <div className="mb-6">
          <Callout title="Setup Authentication">
            <p className="mb-4">
              Create a first user account to enable authentication and secure the system
            </p>
            <Dialog>
              <DialogTrigger asChild>
                <Button variant="primary">Create User</Button>
              </DialogTrigger>
              <DialogContent className="sm:max-w-lg">
                <DialogHeader>
                  <DialogTitle>Create First User</DialogTitle>
                  <DialogDescription>
                    Enter the details for the first administrative user
                  </DialogDescription>
                </DialogHeader>

                <form
                  className="mt-4 space-y-4"
                  onSubmit={async (e) => {
                    e.preventDefault();
                    const formData = new FormData(e.currentTarget);
                    const data = {
                      email: formData.get("email") as string,
                      name: formData.get("name") as string,
                      password: formData.get("password") as string,
                      confirmPassword: formData.get(
                        "confirmPassword",
                      ) as string,
                    };

                    if (data.password !== data.confirmPassword) {
                      toast({
                        title: "Error",
                        description: "Passwords do not match",
                        variant: "error",
                      });
                      return;
                    }

                    try {
                      await queryApi("auth/setup", {
                        method: "POST",
                        body: {
                          email: data.email,
                          name: data.name,
                          password: data.password,
                        },
                      });
                      toast({
                        title: "Success",
                        description: "User created successfully",
                      });
                      await fetchSystemConfig();
                      router.invalidate();
                      setTimeout(() => {
                        navigate({
                          to: "/login",
                          replace: true,
                        });
                      }, 0)
                    } catch (error) {
                      toast({
                        title: "Error",
                        description:
                          error instanceof Error
                            ? error.message
                            : "An error occurred",
                        variant: "error",
                      });
                    }
                  }}
                >
                  <div className="space-y-2">
                    <Label htmlFor="email">Email</Label>
                    <Input
                      id="email"
                      name="email"
                      type="email"
                      required
                      placeholder="admin@example.com"
                    />
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="name">Name</Label>
                    <Input
                      id="name"
                      name="name"
                      type="text"
                      placeholder="Administrator"
                    />
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="password">Password</Label>
                    <Input
                      id="password"
                      name="password"
                      type="password"
                      minLength={8}
                      required
                    />
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="confirmPassword">
                      Confirm Password
                    </Label>
                    <Input
                      id="confirmPassword"
                      name="confirmPassword"
                      type="password"
                      minLength={8}
                      required
                    />
                  </div>

                  <DialogFooter className="mt-6">
                    <DialogClose asChild>
                      <Button type="button" variant="secondary">
                        Cancel
                      </Button>
                    </DialogClose>
                    <Button type="submit">Create User</Button>
                  </DialogFooter>
                </form>
              </DialogContent>
            </Dialog>
          </Callout>
        </div>
      )}

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
                  ? "Invite Link"
                  : (
                    <span>
                      <RiUserAddFill
                        className="size-4 inline mr-1 -mt-1" />
                      Invite User
                    </span>
                  )}
              </DialogTitle>
              <DialogDescription>
                {inviteCode
                  ? ("Share this link with %%").replace(
                    "%%",
                    inviteCode.email,
                  )
                  : "Enter the email address of the user you want to invite"
                }
              </DialogDescription>
            </DialogHeader>

            {inviteCode ? (
              <div>
                <div className="flex items-center gap-2 my-4">
                  <code className="bg-cbga dark:bg-dbga p-2 rounded flex-grow overflow-hidden text-ellipsis">
                    {getInviteLink(inviteCode.code)}
                  </code>
                  <Button
                    onClick={() => {
                      navigator.clipboard.writeText(
                        getInviteLink(inviteCode.code),
                      );
                      toast({
                        title: "Success",
                        description: "Invite link copied to clipboard",
                      });
                    }}
                    variant="light"
                  >
                    Copy
                  </Button>
                </div>
                <DialogFooter>
                  <Button
                    onClick={() => {
                      setShowInviteDialog(false);
                      setInviteCode(null);
                    }}
                  >
                    Close
                  </Button>
                </DialogFooter>
              </div>
            ) : (
              <form
                className="space-y-4 mt-4"
                onSubmit={(e) => {
                  e.preventDefault();
                  const formData = new FormData(e.currentTarget);
                  handleCreateInvite(formData.get("email") as string);
                }}
              >
                <Input
                  id="email"
                  name="email"
                  type="email"
                  required
                  placeholder="user@example.com"
                />

                <DialogFooter>
                  <DialogClose asChild>
                    <Button type="button" variant="secondary">
                      Cancel
                    </Button>
                  </DialogClose>
                  <Button type="submit" className="mb-4 sm:mb-0">Create Invite</Button>
                </DialogFooter>
              </form>
            )}
          </DialogContent>
        </Dialog>
      )}

      <Dialog open={deleteUserDialog !== null} onOpenChange={(open) => !open && setDeleteUserDialog(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Confirm Deletion</DialogTitle>
            <DialogDescription>
              {deleteUserDialog && ("Are you sure you want to delete the user %%?").replace(
                "%%",
                deleteUserDialog.email,
              )}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button onClick={() => setDeleteUserDialog(null)} variant="secondary">
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={() => {
                if (deleteUserDialog) {
                  handleDelete(deleteUserDialog);
                  setDeleteUserDialog(null);
                }
              }}
            >
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={deleteInviteDialog !== null} onOpenChange={(open) => !open && setDeleteInviteDialog(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Confirm Deletion</DialogTitle>
            <DialogDescription>
              {deleteInviteDialog && ("Are you sure you want to delete the invite for %%?").replace(
                "%%",
                deleteInviteDialog.email,
              )}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button onClick={() => setDeleteInviteDialog(null)} variant="secondary">
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={async () => {
                if (deleteInviteDialog) {
                  try {
                    await queryApi(`invites/${deleteInviteDialog.code}`, {
                      method: "DELETE",
                    });
                    toast({
                      title: "Success",
                      description: "Invite deleted successfully",
                    });
                    router.invalidate();
                  } catch (error) {
                    if (isRedirect(error)) {
                      return navigate(error.options);
                    }
                    toast({
                      title: "Error",
                      description: error instanceof Error
                        ? error.message
                        : "An error occurred",
                      variant: "error",
                    });
                  }
                  setDeleteInviteDialog(null);
                }
              }}
            >
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {getSystemConfig().loginRequired && (
        <>
          {data.invites?.length > 0 && (
            <>
              <div className="mb-6">
                <h3 className="text-lg font-semibold mb-3">Pending Invites</h3>
                <TableRoot>
                  <Table>
                    <TableHead>
                      <TableRow>
                        <TableHeaderCell className="text-md text-ctext dark:text-dtext">Email</TableHeaderCell>
                        <TableHeaderCell className="text-md text-ctext dark:text-dtext">Created</TableHeaderCell>
                        <TableHeaderCell className="text-md text-ctext dark:text-dtext">Invite Link</TableHeaderCell>
                        <TableHeaderCell className="text-md text-ctext dark:text-dtext">Actions</TableHeaderCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {data.invites.map((invite) => (
                        <TableRow key={invite.code}>
                          <TableCell className="text-ctext2 dark:text-dtext2">
                            {invite.email}
                          </TableCell>
                          <TableCell className="text-ctext2 dark:text-dtext2">
                            <Tooltip
                              showArrow={false}
                              content={new Date(
                                invite.createdAt,
                              ).toLocaleString()}
                            >
                              {new Date(invite.createdAt).toLocaleDateString()}
                            </Tooltip>
                          </TableCell>
                          <TableCell className="text-ctext2 dark:text-dtext2">
                            {(() => {
                              const inviteState = getInviteState(invite.createdAt, data.inviteValidTimeInSeconds)
                              if (inviteState.isExpired) {
                                return (
                                  <span>Expired</span>
                                )
                              }
                              return (
                                <div className="space-y-1">
                                  <div className="flex items-center gap-2">
                                    <code className="bg-cbg dark:bg-dbg p-1 rounded text-sm overflow-hidden text-ellipsis">
                                      {getInviteLink(invite.code)}
                                    </code>
                                    <Button
                                      variant="light"
                                      onClick={() => {
                                        navigator.clipboard.writeText(getInviteLink(invite.code))
                                        toast({
                                          title: 'Success',
                                          description: 'Invite link copied to clipboard',
                                        })
                                      }}
                                    >
                                      Copy
                                    </Button>
                                  </div>
                                  <div className="text-xs text-ctext2 dark:text-dtext2">
                                    {inviteState.expiresIn}
                                  </div>
                                </div>
                              )
                            })()}
                          </TableCell>
                          <TableCell>
                            <button
                              onClick={() => setDeleteInviteDialog(invite)}
                              className="text-cerr dark:text-derr opacity-90 hover:opacity-100 hover:underline"
                            >
                              Delete
                            </button>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </TableRoot>
              </div>
              <h3 className="text-lg font-semibold mb-3">Users</h3>
            </>
          )}

          <TableRoot>
            <Table>
              <TableHead>
                <TableRow>
                  <TableHeaderCell
                    onClick={() => handleSort("name")}
                    className="text-md text-ctext dark:text-dtext cursor-pointer hover:underline"
                  >
                    Name <SortIcon field="name" />
                  </TableHeaderCell>
                  <TableHeaderCell
                    onClick={() => handleSort("email")}
                    className="text-md text-ctext dark:text-dtext cursor-pointer hover:underline"
                  >
                    Email <SortIcon field="email" />
                  </TableHeaderCell>
                  <TableHeaderCell
                    onClick={() => handleSort("created")}
                    className="text-md text-ctext dark:text-dtext hidden md:table-cell cursor-pointer hover:underline"
                  >
                    Created <SortIcon field="created" />
                  </TableHeaderCell>
                  <TableHeaderCell className="text-md text-ctext dark:text-dtext">
                    Actions
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
                      <Tooltip
                        showArrow={false}
                        content={new Date(user.createdAt).toLocaleString()}
                      >
                        {new Date(user.createdAt).toLocaleDateString()}
                      </Tooltip>
                    </TableCell>
                    <TableCell>
                      <button
                        onClick={() => setDeleteUserDialog(user)}
                        className="text-cerr dark:text-derr opacity-90 hover:opacity-100 hover:underline"
                      >
                        Delete
                      </button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableRoot>
        </>
      )}
    </>
  );
}
