import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { Helmet } from "react-helmet";
import { logout, useAuth } from "../lib/auth";
import { translate } from "../lib/translate";
import { Button } from "../components/tremor/Button";
import { useState } from "react";
import { useToast } from "../hooks/useToast";
import { Toaster } from "../components/tremor/Toaster";

export const Route = createFileRoute("/admin")({
  component: Admin,
});

function Admin() {
  const navigate = useNavigate({ from: "/" });

  return (
    <div className="p-4 max-w-[720px] mx-auto">
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
      </div>

      <div className="mt-8">
        <h2 className="text-xl font-semibold mb-4">
          {translate("Security Settings")}
        </h2>
        <div className="space-y-4">
          <div>
            <h3 className="text-lg font-medium mb-2">JWT Secret</h3>
            <p className="text-gray-600 dark:text-gray-400 mb-4">
              {translate(
                "Reset the JWT secret to invalidate all existing tokens.",
              )}
            </p>
            <ResetJWTButton />
          </div>
        </div>
      </div>
    </div>
  );
}

function ResetJWTButton() {
  const [isResetting, setIsResetting] = useState(false);
  const { toast } = useToast();
  const auth = useAuth();

  const handleReset = async () => {
    setIsResetting(true);
    try {
      const jwt = await auth.getJwt();
      const response = await fetch("/api/admin/reset-jwt-secret", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: jwt,
        },
      });

      if (!response.ok) {
        throw new Error("Failed to reset JWT secret");
      }

      toast({
        title: translate("Success"),
        description: translate("JWT secret reset successfully"),
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
    } finally {
      setIsResetting(false);
    }
  };

  return (
    <Button
      onClick={handleReset}
      disabled={isResetting}
      variant="secondary"
    >
      {isResetting ? translate("Resetting...") : translate("Reset JWT Secret")}
    </Button>
  );
}
