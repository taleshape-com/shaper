import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { Helmet } from "react-helmet";
import { logout, useAuth } from "../lib/auth";
import { translate } from "../lib/translate";
import { Button } from "../components/tremor/Button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeaderCell,
  TableRoot,
  TableRow,
} from "../components/tremor/Table";
import { useCallback, useEffect } from "react";
import { useState } from "react";
import { useToast } from "../hooks/useToast";
import { Toaster } from "../components/tremor/Toaster";

export const Route = createFileRoute("/admin")({
  component: Admin,
});

interface APIKey {
  id: string;
  name: string;
  createdAt: string;
}

interface NewAPIKeyResponse {
  id: string;
  key: string;
}

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
        <h2 className="text-xl font-semibold mb-4">{translate("API Keys")}</h2>
        <APIKeyList />
      </div>

      <div className="mt-12">
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

function APIKeyList() {
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [loading, setLoading] = useState(true);
  const [showNewKeyDialog, setShowNewKeyDialog] = useState(false);
  const [newKey, setNewKey] = useState<NewAPIKeyResponse | null>(null);
  const auth = useAuth();
  const { toast } = useToast();

  const fetchKeys = useCallback(async () => {
    try {
      const jwt = await auth.getJwt();
      const response = await fetch("/api/keys", {
        headers: {
          Authorization: jwt,
        },
      });

      if (!response.ok) {
        throw new Error("Failed to fetch API keys");
      }

      const data = await response.json();
      setKeys(data.keys);
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
      setLoading(false);
    }
  }, [auth, toast]);

  useEffect(() => {
    fetchKeys();
  }, [fetchKeys]);

  const handleDelete = async (key: APIKey) => {
    if (
      !confirm(
        translate('Are you sure you want to delete this API key "%%"?').replace(
          "%%",
          key.name,
        ),
      )
    ) {
      return;
    }

    try {
      const jwt = await auth.getJwt();
      const response = await fetch(`/api/keys/${key.id}`, {
        method: "DELETE",
        headers: {
          Authorization: jwt,
        },
      });

      if (!response.ok) {
        throw new Error("Failed to delete API key");
      }

      toast({
        title: translate("Success"),
        description: translate("API key deleted successfully"),
      });
      fetchKeys();
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
  };

  const handleCreateKey = async (name: string) => {
    try {
      const jwt = await auth.getJwt();
      const response = await fetch("/api/keys", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: jwt,
        },
        body: JSON.stringify({ name }),
      });

      if (!response.ok) {
        throw new Error("Failed to create API key");
      }

      const data = await response.json();
      setNewKey(data);
      fetchKeys();
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
  };

  return (
    <div>
      <Button onClick={() => setShowNewKeyDialog(true)} className="mb-4">
        {translate("New")}
      </Button>

      {loading ? (
        <p>{translate("Loading API keys...")}</p>
      ) : keys.length === 0 ? (
        <p>{translate("No API keys found")}</p>
      ) : (
        <TableRoot>
          <Table>
            <TableHead>
              <TableRow>
                <TableHeaderCell>{translate("Name")}</TableHeaderCell>
                <TableHeaderCell>{translate("Created")}</TableHeaderCell>
                <TableHeaderCell>{translate("Actions")}</TableHeaderCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {keys.map((key) => (
                <TableRow key={key.id}>
                  <TableCell>{key.name}</TableCell>
                  <TableCell>
                    <div title={new Date(key.createdAt).toLocaleString()}>
                      {new Date(key.createdAt).toLocaleDateString()}
                    </div>
                  </TableCell>
                  <TableCell>
                    <Button
                      variant="secondary"
                      onClick={() => handleDelete(key)}
                      className="text-cerr dark:text-derr"
                    >
                      {translate("Delete")}
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableRoot>
      )}

      {showNewKeyDialog && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center">
          <div className="bg-white dark:bg-gray-800 p-6 rounded-lg max-w-md w-full">
            <h3 className="text-lg font-medium mb-4">
              {translate("Create New API Key")}
            </h3>
            {newKey ? (
              <div>
                <p className="mb-2">{translate("Your new API key:")}</p>
                <div className="flex items-center gap-2 mb-4">
                  <code className="bg-gray-100 dark:bg-gray-700 p-2 rounded flex-grow max-w-80 overflow-clip text-ellipsis">
                    {newKey.key}
                  </code>
                  <Button
                    onClick={() => {
                      navigator.clipboard.writeText(newKey.key);
                      toast({
                        title: translate("Success"),
                        description: translate("API key copied to clipboard"),
                      });
                    }}
                    variant="secondary"
                  >
                    {translate("Copy")}
                  </Button>
                </div>
                <p className="text-sm text-red-500 mb-4">
                  {translate(
                    "Make sure to copy this key now. You won't be able to see it again!",
                  )}
                </p>
                <Button
                  onClick={() => {
                    setShowNewKeyDialog(false);
                    setNewKey(null);
                  }}
                >
                  {translate("Close")}
                </Button>
              </div>
            ) : (
              <form
                onSubmit={(e) => {
                  e.preventDefault();
                  const formData = new FormData(e.currentTarget);
                  handleCreateKey(formData.get("name") as string);
                }}
              >
                <input
                  type="text"
                  name="name"
                  placeholder={translate("Key name")}
                  className="w-full p-2 border rounded mb-4 dark:bg-gray-700 dark:border-gray-600"
                  required
                />
                <div className="flex justify-end gap-2">
                  <Button
                    type="button"
                    variant="secondary"
                    onClick={() => setShowNewKeyDialog(false)}
                  >
                    {translate("Cancel")}
                  </Button>
                  <Button type="submit">{translate("Create")}</Button>
                </div>
              </form>
            )}
          </div>
        </div>
      )}
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
    <Button onClick={handleReset} disabled={isResetting} variant="secondary">
      {isResetting ? translate("Resetting...") : translate("Reset JWT Secret")}
    </Button>
  );
}
