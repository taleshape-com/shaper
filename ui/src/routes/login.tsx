// SPDX-License-Identifier: MPL-2.0

import * as React from "react";
import {
  createFileRoute,
  useRouter,
  redirect,
} from "@tanstack/react-router";
import { ErrorComponent } from "../components/ErrorComponent";
import { z } from "zod";
import { ErrorComponentProps } from "@tanstack/react-router";
import { Input } from "../components/tremor/Input";
import { Helmet } from "react-helmet";
import { RiCheckLine, RiFileCopyLine } from "@remixicon/react";
import { useAuth, testLogin } from "../lib/auth";
import { getSystemConfig } from "../lib/system";
import { Button } from "../components/tremor/Button";
import { copyToClipboard } from "../lib/utils";

export const Route = createFileRoute("/login")({
  validateSearch: z.object({
    redirect: z.string().optional(),
    logout: z.boolean().or(z.string()).optional(),
  }),
  loaderDeps: ({ search: { redirect, logout } }) => ({
    redirectUrl: redirect,
    logout: !!logout,
  }),
  loader: async ({
    deps: { redirectUrl, logout },
  }) => {
    if (logout) {
      return;
    }
    if (await testLogin()) {
      throw redirect({
        to: redirectUrl || "/",
      });
    }
    const config = getSystemConfig();
    if (config.ssoLoginUrl) {
      const ssoUrl = new URL(config.ssoLoginUrl, window.location.href);
      const targetRedirect = redirectUrl || window.shaper.defaultBaseUrl || "/";
      const finalRedirect = new URL(targetRedirect, window.location.href).toString();
      ssoUrl.searchParams.set("redirect", finalRedirect);
      window.location.href = ssoUrl.toString();
    }
  },
  component: LoginComponent,
  errorComponent: LoginErrorComponent as any,
});

function LoginErrorComponent ({ error }: ErrorComponentProps) {
  return <ErrorComponent error={error} />;
}

function LoginComponent () {
  const auth = useAuth();
  const router = useRouter();
  const search = Route.useSearch();
  const [email, setEmail] = React.useState("");
  const [password, setPassword] = React.useState("");
  const [err, setError] = React.useState("");
  const [isLoggingIn, setIsLoggingIn] = React.useState(false);

  const [copied, setCopied] = React.useState(false);

  const handleCopy = async () => {
    const success = await copyToClipboard(err);
    if (success) {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const onSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setIsLoggingIn(true);
    setError("");
    const ok = await auth.login(email, password);
    if (ok) {
      router.history.push(search.redirect || window.shaper.defaultBaseUrl);
    } else {
      setError("Invalid email or password");
    }
    setIsLoggingIn(false);
  };

  const config = getSystemConfig();

  if (config.ssoLoginUrl) {
    const handleSSOLogin = () => {
      const ssoUrl = new URL(config.ssoLoginUrl!, window.location.href);
      const targetRedirect = search.redirect || window.shaper.defaultBaseUrl || "/";
      const finalRedirect = new URL(targetRedirect, window.location.href).toString();
      ssoUrl.searchParams.set("redirect", finalRedirect);
      window.location.href = ssoUrl.toString();
    };

    return (
      <div className="flex items-center justify-center h-screen">
        <div className="px-8 py-10 w-96 rounded-xl border border-ctext/10 dark:border-dtext/10 bg-cbgs dark:bg-dbgs shadow-xl space-y-6 text-center">
          <Helmet>
            <title>Login</title>
            <meta name="description" content="Login with SSO to continue" />
          </Helmet>
          <div className="space-y-2">
            <h1 className="text-2xl font-bold font-display">Logged Out</h1>
            <p className="text-sm text-ctext/60 dark:text-dtext/60">
              You have been successfully logged out of Shaper. Click below to sign in again.
            </p>
          </div>
          <Button
            onClick={handleSSOLogin}
            variant="primary"
            className="w-full py-2.5 font-semibold text-white bg-indigo-600 hover:bg-indigo-700"
          >
            Login with SSO
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="flex items-center justify-center h-screen">
      <div className="px-6 pt-2 pb-10">
        <Helmet>
          <title>Login</title>
          <meta name="description" content="Login to continue" />
        </Helmet>
        <form
          onSubmit={onSubmit}
          className="space-y-4 w-80 "
          name="login"
          autoComplete="on"
        >
          <h1 className="text-xl font-semibold text-center">Welcome</h1>
          <Input
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="Email"
            type="email"
            autoFocus
            name="email"
            id="email"
            autoComplete="username email"
            required
            disabled={isLoggingIn}
          />
          <Input
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="Password"
            type="password"
            name="current-password"
            id="current-password"
            autoComplete="current-password"
            required
            disabled={isLoggingIn}
          />
          <Button
            type="submit"
            variant="primary"
            disabled={!email || !password || isLoggingIn}
            className="w-full py-2"
          >
            {isLoggingIn ? "Logging in..." : "Login"}
          </Button>
        </form>
        {err && (
          <div className="mt-4 text-red-500 text-sm flex items-center justify-between">
            <span>{err}</span>
            <button
              onClick={handleCopy}
              className="ml-2 text-red-400 hover:text-red-600 transition-colors"
              title="Copy error message"
            >
              {copied ? (
                <RiCheckLine className="size-4" />
              ) : (
                <RiFileCopyLine className="size-4" />
              )}
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
