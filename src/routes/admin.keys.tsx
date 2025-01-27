import { Button } from "../components/tremor/Button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../components/tremor/Dialog";
import { Input } from "../components/tremor/Input";
import { Label } from "../components/tremor/Label";
import { useState } from "react";
import {
  createFileRoute,
  isRedirect,
  useNavigate,
} from "@tanstack/react-router";
import { useToast } from "../hooks/useToast";
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
import { useCallback, useEffect } from "react";
import { useQueryApi } from "../hooks/useQueryApi";

interface APIKey {
  id: string;
  name: string;
  createdAt: string;
}

interface NewAPIKeyResponse {
  id: string;
  key: string;
}

export const Route = createFileRoute("/admin/keys")({
  component: Admin,
});

function Admin() {
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [loading, setLoading] = useState(true);
  const [showNewKeyDialog, setShowNewKeyDialog] = useState(false);
  const [newKey, setNewKey] = useState<NewAPIKeyResponse | null>(null);
  const queryApi = useQueryApi();
  const navigate = useNavigate({ from: "/admin" });
  const { toast } = useToast();

  const fetchKeys = useCallback(async () => {
    try {
      const data = await queryApi("/api/keys");
      setKeys(data.keys);
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
    } finally {
      setLoading(false);
    }
  }, [queryApi, toast, navigate]);

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
      await queryApi(`/api/keys/${key.id}`, {
        method: "DELETE",
      });
      toast({
        title: translate("Success"),
        description: translate("API key deleted successfully"),
      });
      fetchKeys();
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

  const handleCreateKey = async (name: string) => {
    try {
      const data = await queryApi("/api/keys", {
        method: "POST",
        body: { name },
      });
      console.log("hellp");
      setNewKey(data);
      fetchKeys();
    } catch (error) {
      console.log("er", error);
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
    <div>
      <h2 className="text-xl font-semibold mb-4">{translate("API Keys")}</h2>

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
                      variant="destructive"
                      onClick={() => handleDelete(key)}
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

      <Dialog
        open={showNewKeyDialog}
        onOpenChange={(open) => {
          setShowNewKeyDialog(open);
          if (!open) setNewKey(null);
        }}
      >
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>
              {newKey
                ? translate("API Key Created")
                : translate("Create New API Key")}
            </DialogTitle>
            {!newKey && (
              <DialogDescription>
                {translate("Enter a name to identify this API key")}
              </DialogDescription>
            )}
          </DialogHeader>

          {newKey ? (
            <div className="space-y-4">
              <div>
                <Label>{translate("Your new API key:")}</Label>
                <div className="flex items-center gap-2 mt-2">
                  <code className="bg-cbga dark:bg-db p-2 rounded flex-grow overflow-hidden text-ellipsis">
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
                    variant="primary"
                  >
                    {translate("Copy")}
                  </Button>
                </div>
              </div>

              <p className="text-sm text-cerr dark:text-derr">
                {translate(
                  "Make sure to copy this key now. You won't be able to see it again!",
                )}
              </p>

              <DialogFooter>
                <Button
                  variant="light"
                  onClick={() => {
                    setShowNewKeyDialog(false);
                    setNewKey(null);
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
                handleCreateKey(formData.get("name") as string);
              }}
            >
              <Input
                id="name"
                name="name"
                placeholder={translate("Enter key name")}
                required
                autoFocus
              />

              <DialogFooter>
                <DialogClose asChild>
                  <Button type="button" variant="destructive">
                    {translate("Cancel")}
                  </Button>
                </DialogClose>
                <Button type="submit">{translate("Create Key")}</Button>
              </DialogFooter>
            </form>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
