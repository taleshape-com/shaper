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
import { useState } from "react";
import { Button } from "../components/tremor/Button";
import { useToast } from "../hooks/useToast";
import { useRouter } from "@tanstack/react-router";
import { useQueryApi } from "../hooks/useQueryApi";
import { useAuth } from "../lib/auth";
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
    expiresIn = translate('Expires in %% days').replace('%%', days.toString())
  } else if (hours > 0) {
    expiresIn = translate('Expires in %% hours').replace('%%', hours.toString())
  } else {
    expiresIn = translate('Expires soon')
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
  const data = Route.useLoaderData();
  const auth = useAuth();
  const { sort, order } = Route.useSearch();
  const navigate = useNavigate({ from: "/admin" });
  const [showInviteDialog, setShowInviteDialog] = useState(false);
  const [inviteCode, setInviteCode] = useState<{
    code: string;
    email: string;
  } | null>(null);
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
      router.invalidate();
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
        {auth.loginRequired && (
          <Button onClick={() => setShowInviteDialog(true)}>
            {translate("Invite User")}
          </Button>
        )}
      </div>

      {auth.loginRequired === false && (
        <div className="mb-6">
          <Callout title={translate("Setup Authentication")}>
            <p className="mb-4">
              {translate(
                "Create a first user account to enable authentication and secure the system",
              )}
            </p>
            <Dialog>
              <DialogTrigger asChild>
                <Button variant="primary">{translate("Create User")}</Button>
              </DialogTrigger>
              <DialogContent className="sm:max-w-lg">
                <DialogHeader>
                  <DialogTitle>{translate("Create First User")}</DialogTitle>
                  <DialogDescription>
                    {translate(
                      "Enter the details for the first administrative user",
                    )}
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
                        title: translate("Error"),
                        description: translate("Passwords do not match"),
                        variant: "error",
                      });
                      return;
                    }

                    try {
                      await queryApi("/api/auth/setup", {
                        method: "POST",
                        body: {
                          email: data.email,
                          name: data.name,
                          password: data.password,
                        },
                      });
                      toast({
                        title: translate("Success"),
                        description: translate("User created successfully"),
                      });
                      auth.setLoginRequired(true);
                      navigate({
                        to: "/login",
                        replace: true,
                      });
                    } catch (error) {
                      toast({
                        title: translate("Error"),
                        description:
                          error instanceof Error
                            ? error.message
                            : translate("An error occurred"),
                        variant: "error",
                      });
                    }
                  }}
                >
                  <div className="space-y-2">
                    <Label htmlFor="email">{translate("Email")}</Label>
                    <Input
                      id="email"
                      name="email"
                      type="email"
                      required
                      placeholder="admin@example.com"
                    />
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="name">{translate("Name")}</Label>
                    <Input
                      id="name"
                      name="name"
                      type="text"
                      placeholder={translate("Administrator")}
                    />
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="password">{translate("Password")}</Label>
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
                      {translate("Confirm Password")}
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
                        {translate("Cancel")}
                      </Button>
                    </DialogClose>
                    <Button type="submit">{translate("Create User")}</Button>
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
                  ? translate("Invite Link")
                  : translate("Invite User")}
              </DialogTitle>
              <DialogDescription>
                {inviteCode
                  ? translate("Share this link with %%").replace(
                    "%%",
                    inviteCode.email,
                  )
                  : translate(
                    "Enter the email address of the user you want to invite",
                  )}
              </DialogDescription>
            </DialogHeader>

            {inviteCode ? (
              <div>
                <div className="flex items-center gap-2 my-4">
                  <code className="bg-cbga dark:bg-db p-2 rounded flex-grow overflow-hidden text-ellipsis">
                    {getInviteLink(inviteCode.code)}
                  </code>
                  <Button
                    onClick={() => {
                      navigator.clipboard.writeText(
                        getInviteLink(inviteCode.code),
                      );
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

      {auth.loginRequired && (
        <>
          {data.invites?.length > 0 && (
            <>
              <div className="mb-6">
                <h3 className="text-lg font-semibold mb-3">
                  {translate("Pending Invites")}
                </h3>
                <TableRoot>
                  <Table>
                    <TableHead>
                      <TableRow>
                        <TableHeaderCell className="text-md text-ctext dark:text-dtext">
                          {translate("Email")}
                        </TableHeaderCell>
                        <TableHeaderCell className="text-md text-ctext dark:text-dtext">
                          {translate("Created")}
                        </TableHeaderCell>
                        <TableHeaderCell className="text-md text-ctext dark:text-dtext">
                          {translate("Invite Link")}
                        </TableHeaderCell>
                        <TableHeaderCell className="text-md text-ctext dark:text-dtext">
                          {translate("Actions")}
                        </TableHeaderCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {data.invites.map((invite) => (
                        <TableRow key={invite.code}>
                          <TableCell className="text-ctext2 dark:text-dtext2">
                            {invite.email}
                          </TableCell>
                          <TableCell className="text-ctext2 dark:text-dtext2">
                            <div
                              title={new Date(
                                invite.createdAt,
                              ).toLocaleString()}
                            >
                              {new Date(invite.createdAt).toLocaleDateString()}
                            </div>
                          </TableCell>
                          <TableCell className="text-ctext2 dark:text-dtext2">
                            {(() => {
                              const inviteState = getInviteState(invite.createdAt, data.inviteValidTimeInSeconds)
                              if (inviteState.isExpired) {
                                return (
                                  <span>
                                    {translate('Expired')}
                                  </span>
                                )
                              }
                              return (
                                <div className="space-y-1">
                                  <div className="flex items-center gap-2">
                                    <code className="bg-cbga dark:bg-db p-1 rounded text-sm flex-grow overflow-hidden text-ellipsis">
                                      {getInviteLink(invite.code)}
                                    </code>
                                    <Button
                                      variant="secondary"
                                      onClick={() => {
                                        navigator.clipboard.writeText(getInviteLink(invite.code))
                                        toast({
                                          title: translate('Success'),
                                          description: translate('Invite link copied to clipboard'),
                                        })
                                      }}
                                    >
                                      {translate('Copy')}
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
                              onClick={async () => {
                                if (
                                  !window.confirm(
                                    translate(
                                      "Are you sure you want to delete the invite for %%?",
                                    ).replace("%%", invite.email),
                                  )
                                ) {
                                  return;
                                }
                                try {
                                  await queryApi(
                                    `/api/invites/${invite.code}`,
                                    {
                                      method: "DELETE",
                                    },
                                  );
                                  toast({
                                    title: translate("Success"),
                                    description: translate(
                                      "Invite deleted successfully",
                                    ),
                                  });
                                  router.invalidate();
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
                              }}
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
              </div>
              <h3 className="text-lg font-semibold mb-3">
                {translate("Users")}
              </h3>
            </>
          )}

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
        </>
      )}
    </>
  );
}
