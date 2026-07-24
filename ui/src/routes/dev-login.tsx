// SPDX-License-Identifier: MPL-2.0

import { useState } from "react";
import { createFileRoute, redirect } from "@tanstack/react-router";
import { z } from "zod";
import { Helmet } from "react-helmet";

import { MenuProvider } from "../components/providers/MenuProvider";
import { MenuTrigger } from "../components/MenuTrigger";
import { Button } from "../components/tremor/Button";
import { Card } from "../components/tremor/Card";
import { localStorageTokenKey, localStorageJwtKey, testLogin } from "../lib/auth";

export const Route = createFileRoute("/dev-login")({
  validateSearch: z.object({
    port: z.coerce.number().int().positive().optional(),
  }),
  loaderDeps: ({ search: { port } }) => ({ port }),
  loader: async ({ deps: { port } }) => {
    if (!(await testLogin())) {
      const redirectTarget =
        port != null ? `/dev-login?port=${port}` : "/dev-login";
      throw redirect({
        to: "/login",
        search: {
          redirect: redirectTarget,
        },
      });
    }
  },
  component: DevLoginPage,
});

type Status = "idle" | "sending" | "success" | "error";

function DevLoginPage () {
  const { port } = Route.useSearch();
  const [status, setStatus] = useState<Status>("idle");
  const [message, setMessage] = useState("");

  const sendToken = async () => {
    if (!port) {
      setStatus("error");
      setMessage("Missing port information. Restart the CLI to try again.");
      return;
    }

    const currentJwt = window.localStorage.getItem(localStorageJwtKey);
    const sessionToken = window.localStorage.getItem(localStorageTokenKey);

    setStatus("sending");
    setMessage("Generating secure CLI token...");
    try {
      const headers: Record<string, string> = {
        "Content-Type": "application/json",
      };
      if (currentJwt) {
        headers["Authorization"] = currentJwt.startsWith("Bearer ") ? currentJwt : `Bearer ${currentJwt}`;
      }

      const tokenResp = await fetch(`${window.shaper.defaultBaseUrl}api/auth/token`, {
        method: "POST",
        headers,
        body: JSON.stringify({
          token: sessionToken || undefined,
          longLived: true,
        }),
      });

      if (!tokenResp.ok) {
        const text = await tokenResp.text();
        throw new Error(`Failed to generate CLI token: ${text || tokenResp.statusText}`);
      }

      const { jwt: cliJwt } = await tokenResp.json();
      if (!cliJwt) {
        throw new Error("Failed to generate CLI token: response was missing the jwt field.");
      }

      const response = await fetch(`http://localhost:${port}/token`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ token: cliJwt }),
      });
      if (!response.ok) {
        const text = await response.text();
        throw new Error(text || response.statusText);
      }

      setStatus("success");
      setMessage("Authentication confirmed. You can close this window.");
    } catch (err) {
      setStatus("error");
      setMessage(
        err instanceof Error
          ? err.message
          : "Failed to send authentication token.",
      );
    }
  };

  return (
    <MenuProvider>
      <Helmet>
        <title>Shaper Dev Login</title>
        <meta
          name="description"
          content="Authorize the shaper dev CLI to access your session."
        />
      </Helmet>
      <div className="px-4 pb-4 min-h-dvh flex flex-col">
        <div className="flex">
          <MenuTrigger className="pr-1.5 py-3 -ml-1.5" />
          <h1 className="text-2xl font-semibold font-display flex-grow pb-2 pt-2.5">
            Login
          </h1>
        </div>

        <div className="bg-cbgs dark:bg-dbgs rounded-md shadow flex-grow flex items-center justify-center">
          <Card className="max-w-lg w-full m-6 p-6 space-y-4">
            <div className="space-y-6">
              <p className="text-xl font-semibold">Authorize Shaper Dev Mode</p>
              <p className="text-ctext2 dark:text-dtext2 pb-4">
                Confirm to give the Shaper Dev CLI access to your user account.
              </p>
              {
                port ? (
                  <div className="rounded-md border border-cborder dark:border-dborder bg-cbg-secondary dark:bg-dbg-secondary p-4 text-sm">
                    <p className="font-semibold">
                      CLI port: <span className="font-mono text-base">{port}</span>
                    </p>
                    <p className="text-ctext2 dark:text-dtext2">
                      Make sure the number here matches the port shown in your terminal before you send the token.
                    </p>
                  </div>
                ) : (
                  <div className="rounded-md border border-cborder dark:border-dborder bg-cbg-secondary dark:bg-dbg-secondary p-4 text-sm text-cerr dark:text-derr">
                    Unable to detect the CLI port. Restart the CLI to try again.
                  </div>
                )
              }
              {
                status !== "success" && (
                  <Button
                    onClick={sendToken}
                    disabled={status === "sending" || !port}
                    className="w-full font-semibold p-2"
                  >
                    {status === "sending" ? "Confirming..." : "Confirm"}
                  </Button>
                )
              }
              {message && (
                <p
                  className={
                    status === "error"
                      ? "text-sm text-cerr dark:text-derr"
                      : "text-sm text-green-600 dark:text-green-400 pb-5"
                  }
                >
                  {message}
                </p>
              )}
            </div>
          </Card>
        </div>
      </div>
    </MenuProvider>
  );
}
