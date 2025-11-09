// SPDX-License-Identifier: MPL-2.0

import { RouterProvider } from "@tanstack/react-router";
import { useAuth } from "./lib/auth";
import { useQueryApi } from "./hooks/useQueryApi";

export function App ({ router }: { router: any }) {
  const auth = useAuth();
  const queryApi = useQueryApi();
  return <RouterProvider router={router} context={{ auth, queryApi }} />;
}
