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
import { RiAddFill, RiKeyFill, RiTableFill } from "@remixicon/react";

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
      <div className="flex justify-between items-center mb-4">
        <h2 className="text-xl font-semibold mb-4">
          <RiKeyFill className="size-4 inline mr-1 -mt-0.5" />
          {translate("API Keys")}
        </h2>
        <Button onClick={() => setShowNewKeyDialog(true)}>
          <RiAddFill className="-ml-1 mr-0.5 size-4 shrink-0" aria-hidden={true} />
          {translate("New")}
        </Button>
      </div>

      {loading ? (
        <p>{translate("Loading API keys...")}</p>
      ) : keys.length === 0 ? (
        <div className="mt-4 flex flex-col h-44 items-center justify-center rounded-sm p-4 bg-cbg text-center">
          <RiTableFill className="text-ctext2 dark:text-dtext2 mx-auto h-7 w-7" aria-hidden={true} />
          <p className="mt-2 text-ctext2 dark:text-dtext2 font-medium">
            {translate("No API keys found")}
          </p>
        </div>
      ) : (
        <TableRoot>
          <Table>
            <TableHead>
              <TableRow>
                <TableHeaderCell className="text-md text-ctext dark:text-dtext">{translate("Name")}</TableHeaderCell>
                <TableHeaderCell className="text-md text-ctext dark:text-dtext hidden md:table-cell">{translate("Created")}</TableHeaderCell>
                <TableHeaderCell>{translate("Actions")}</TableHeaderCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {keys.map((key) => (
                <TableRow key={key.id}>
                  <TableCell className="font-medium text-ctext dark:text-dtext">{key.name}</TableCell>
                  <TableCell className="font-medium text-ctext dark:text-dtext hidden md:table-cell">
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
                <Button type="submit" className="mb-4 sm:mb-0">{translate("Create Key")}</Button>
              </DialogFooter>
            </form>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
