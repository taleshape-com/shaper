// SPDX-License-Identifier: MPL-2.0

import * as React from "react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { z } from "zod";
import { translate } from "../lib/translate";
import { Helmet } from "react-helmet";
import { Button } from "../components/tremor/Button";
import { Input } from "../components/tremor/Input";
import { Label } from "../components/tremor/Label";
import { useToast } from "../hooks/useToast";

interface Invite {
  code: string;
  email: string;
  createdAt: string;
}

export const Route = createFileRoute("/signup")({
  validateSearch: z.object({
    code: z.string().optional(),
  }),
  loaderDeps: ({ search: { code } }) => ({ code }),
  loader: async ({ deps: { code } }) => {
    if (!code) {
      return null;
    }
    try {
      const response = await fetch(`${window.shaper.defaultBaseUrl}api/invites/${code}`, {
        headers: {
          "Content-Type": "application/json",
        },
      })
      const data = await response.json();
      if (response.status !== 200) {
        return { error: data.error } as { invite?: Invite; error?: string };
      }
      return { invite: data } as { invite?: Invite; error?: string };
    } catch (error) {
      console.error(error);
      return null;
    }
  },
  component: SignupComponent,
});

function SignupComponent() {
  const navigate = useNavigate({ from: "/signup" });
  const { toast } = useToast();
  const data = Route.useLoaderData();
  const { code } = Route.useSearch();

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const formData = new FormData(e.currentTarget);

    const password = formData.get("password") as string;
    const confirmPassword = formData.get("confirmPassword") as string;

    if (password !== confirmPassword) {
      toast({
        title: translate("Error"),
        description: translate("Passwords do not match"),
        variant: "error",
      });
      return;
    }

    try {
      const response = await fetch(`${window.shaper.defaultBaseUrl}api/invites/${code}/claim`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          name: formData.get("name"),
          password: password,
        }),
      });
      const data = await response.json();
      if (response.status !== 200) {
        throw new Error(data.error);
      }

      toast({
        title: translate("Success"),
        description: translate("Account created successfully"),
      });

      navigate({ to: "/login", replace: true });
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

  if (!code) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-cbg dark:bg-dbg">
        <div className="w-full max-w-md space-y-4 p-6">
          <h1 className="text-2xl font-semibold text-center text-ctext dark:text-dtext">
            {translate("Invalid Invite")}
          </h1>
          <p className="text-center text-ctext2 dark:text-dtext2">
            {translate("No invite code provided")}
          </p>
          <div className="text-center">
            <Button asChild>
              <a href="/login">{translate("Go to Login")}</a>
            </Button>
          </div>
        </div>
      </div>
    );
  }

  if (data?.error) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-cbg dark:bg-dbg">
        <div className="w-full max-w-md space-y-4 p-6">
          <h1 className="text-2xl font-semibold text-center text-ctext dark:text-dtext">
            {translate("Invalid Invite")}
          </h1>
          {data.error === "invite has expired" ? (
            <p className="text-center text-ctext2 dark:text-dtext2">
              {translate("This invite has expired")}
            </p>
          ) : (
            <p className="text-center text-ctext2 dark:text-dtext2">
              {translate(
                "This invite code is invalid or has already been used",
              )}
            </p>
          )}
          <div className="text-center">
            <Button asChild>
              <a href="/login">{translate("Go to Login")}</a>
            </Button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-cbg dark:bg-dbg">
      <Helmet>
        <title>{translate("Sign Up")}</title>
      </Helmet>

      <div className="w-full max-w-md space-y-8 p-6">
        <div>
          <h1 className="text-2xl font-semibold text-center text-ctext dark:text-dtext">
            {translate("Create Account")}
          </h1>
          <p className="mt-2 text-center text-ctext2 dark:text-dtext2">
            {translate("Sign up with invite for")} <span className="font-semibold">{data?.invite?.email}</span>
          </p>
        </div>

        <form
          className="mt-8 space-y-6"
          onSubmit={handleSubmit}
          name="signup"
          autoComplete="on"
        >
          <div className="space-y-4">
            <div>
              <Label htmlFor="name">{translate("Name")}</Label>
              <Input
                id="name"
                name="name"
                type="text"
                autoComplete="name"
                required
                className="mt-1"
              />
            </div>

            <div>
              <Label htmlFor="password">{translate("Password")}</Label>
              <Input
                id="password"
                name="password"
                type="password"
                autoComplete="new-password"
                required
                minLength={8}
                className="mt-1"
              />
            </div>

            <div>
              <Label htmlFor="confirmPassword">
                {translate("Confirm Password")}
              </Label>
              <Input
                id="confirmPassword"
                name="confirmPassword"
                type="password"
                autoComplete="new-password"
                required
                minLength={8}
                className="mt-1"
              />
            </div>
          </div>

          <div>
            <Button type="submit" className="w-full">
              {translate("Create Account")}
            </Button>
          </div>
        </form>

        <p className="text-center text-sm text-ctext2 dark:text-dtext2">
          {translate("Already have an account?")}{" "}
          <br />
          <a href="/login" className="text-cpri dark:text-dpri underline hover:font-semibold">
            {translate("Sign in")}
          </a>
        </p>
      </div>
    </div>
  );
}
