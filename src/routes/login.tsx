import * as React from "react";
import {
  createFileRoute,
  useRouter,
  redirect,
  ErrorComponent,
} from "@tanstack/react-router";
import { z } from "zod";
import { login, testCookie } from "../lib/auth";
import { ErrorComponentProps } from "@tanstack/react-router";

export const Route = createFileRoute("/login")({
  validateSearch: z.object({
    redirect: z.string().optional(),
  }),
  loaderDeps: ({ search: { redirect } }) => ({
    redirectUrl: redirect,
  }),
  loader: async ({ deps: { redirectUrl } }) => {
    const isLoggedIn = await testCookie();
    if (isLoggedIn) {
      throw redirect({
        to: redirectUrl || "/",
      });
    }
  },
  errorComponent: LoginErrorComponent as any,
}).update({
  component: LoginComponent,
});

function LoginErrorComponent({ error }: ErrorComponentProps) {
  return <ErrorComponent error={error} />;
}

function LoginComponent() {
  const router = useRouter();
  const search = Route.useSearch();
  const [token, setToken] = React.useState("");
  const [err, setError] = React.useState("");
  const [isLoggingIn, setIsLoggingIn] = React.useState(false);

  const onSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setIsLoggingIn(true);
    setError("");
    const err = await login(token);
    if (err) {
      setError(err);
    } else {
      router.history.push(search.redirect || "/");
    }
    setIsLoggingIn(false);
  };

  return (
    <div className="p-2">
      <div>Login Required:</div>
      <div className="h-2" />
      <form onSubmit={onSubmit} className="flex gap-2">
        <input
          value={token}
          onChange={(e) => setToken(e.target.value)}
          placeholder="Token"
          autoFocus
          className="border p-1 px-2 rounded"
          disabled={isLoggingIn}
        />
        <button
          type="submit"
          className="text-sm bg-blue-500 text-white border inline-block py-1 px-2 rounded"
          disabled={token === "" || isLoggingIn}
        >
          Login
        </button>
      </form>
      {err !== "" && <div className="text-red-500">{err}</div>}
    </div>
  );
}
