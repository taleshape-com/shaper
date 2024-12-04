import { RouterProvider } from "@tanstack/react-router";
import { useAuth } from './lib/auth'

export function App({ router }: { router: any }) {
  const auth = useAuth()
  return <RouterProvider router={router} context={{ auth }} />
}


