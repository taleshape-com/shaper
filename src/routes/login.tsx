import * as React from "react";
import {
  createFileRoute,
  useRouter,
  redirect,
  ErrorComponent,
} from "@tanstack/react-router";
import { z } from "zod";
import { ErrorComponentProps } from "@tanstack/react-router";
import { Input } from "../components/tremor/Input";
import { Helmet } from "react-helmet";
import { useAuth } from "../lib/auth";
import { Button } from "../components/tremor/Button";

export const Route = createFileRoute("/login")({
  validateSearch: z.object({
    redirect: z.string().optional(),
  }),
  loaderDeps: ({ search: { redirect } }) => ({
    redirectUrl: redirect,
  }),
  loader: async ({
    deps: { redirectUrl },
    context: {
      auth: { testLogin },
    },
  }) => {
    if (await testLogin()) {
      throw redirect({
        to: redirectUrl || "/",
      });
    }
  },
  component: LoginComponent,
  errorComponent: LoginErrorComponent as any,
});

function LoginErrorComponent({ error }: ErrorComponentProps) {
  return <ErrorComponent error={error} />;
}

function LoginComponent() {
  const auth = useAuth();
  const router = useRouter();
  const search = Route.useSearch();
  const [email, setEmail] = React.useState("");
  const [password, setPassword] = React.useState("");
  const [err, setError] = React.useState("");
  const [isLoggingIn, setIsLoggingIn] = React.useState(false);

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
        {err && <div className="mt-4 text-red-500 text-sm">{err}</div>}
      </div>
    </div>
  );
}
