import { createFileRoute, Link, Outlet, useNavigate } from "@tanstack/react-router";
import { Helmet } from "react-helmet";
import { logout, useAuth } from "../lib/auth";
import { translate } from "../lib/translate";
import { Button } from "../components/tremor/Button";
import { useState } from "react";
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
import { useToast } from "../hooks/useToast";
import { Toaster } from "../components/tremor/Toaster";
import { Tabs, TabsList, TabsTrigger } from "../components/tremor/Tabs";

export const Route = createFileRoute("/admin")({
  component: Admin,
});

function Admin() {
  const [showAuthSetup, setShowAuthSetup] = useState(true);
  const { toast } = useToast();
  const navigate = useNavigate({ from: "/" });
  const auth = useAuth();

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
      <div className="p-4 max-w-[1200px] mx-auto">
        <Helmet>
          <title>{translate("Admin")}</title>
          <meta name="description" content="Admin Settings" />
        </Helmet>
        <Toaster />
        <div className="mb-4 flex items-center">
          <h1 className="text-3xl font-semibold font-display flex-grow">
            {translate("Admin")}
          </h1>
          <Button asChild className="h-fit">
            <Link to="/">{translate("Overview")}</Link>
          </Button>
          {auth.loginRequired && (
            <Button
              onClick={() => {
                logout();
                navigate({
                  to: "/login",
                  replace: true,
                });
              }}
              variant="secondary"
              className="h-fit ml-3"
            >
              {translate("Logout")}
            </Button>
          )}
        </div>

        {showAuthSetup && (
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
                        const response = await fetch("/api/auth/setup", {
                          method: "POST",
                          headers: {
                            "Content-Type": "application/json",
                          },
                          body: JSON.stringify({
                            email: data.email,
                            name: data.name,
                            password: data.password,
                          }),
                        });

                        if (!response.ok) {
                          const errorData = await response.json();
                          throw new Error(
                            errorData.error || "Failed to create user",
                          );
                        }

                        toast({
                          title: translate("Success"),
                          description: translate("User created successfully"),
                        });
                        setShowAuthSetup(false);
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

        <div className="bg-white dark:bg-gray-800 rounded-lg shadow">
          <div className="px-6 pt-6">
            <Tabs defaultValue="general" className="w-full">
              <TabsList>
                <TabsTrigger value="general" asChild>
                  <Link to="/admin">{translate("General")}</Link>
                </TabsTrigger>
                <TabsTrigger value="users" asChild>
                  <Link to="/admin/users">{translate("Users")}</Link>
                </TabsTrigger>
              </TabsList>
            </Tabs>
          </div>

          <div className="p-6">
            <Outlet />
          </div>
        </div>
      </div>
    </div>
  );
}
