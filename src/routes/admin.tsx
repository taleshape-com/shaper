import {
  createFileRoute,
  Link,
  Outlet,
  useNavigate,
} from "@tanstack/react-router";
import { Helmet } from "react-helmet";
import { logout, useAuth } from "../lib/auth";
import { translate } from "../lib/translate";
import { Button } from "../components/tremor/Button";
import { Tabs, TabsList, TabsTrigger } from "../components/tremor/Tabs";

export const Route = createFileRoute("/admin")({
  component: Admin,
});

function Admin() {
  const navigate = useNavigate({ from: "/" });
  const auth = useAuth();

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
      <div className="p-4 max-w-[1200px] mx-auto">
        <Helmet>
          <title>{translate("Admin")}</title>
          <meta name="description" content="Admin Settings" />
        </Helmet>
        <div className="mb-4 flex items-center">
          <h1 className="text-3xl font-semibold font-display flex-grow">
            {translate("Admin")}
          </h1>
          <Button asChild className="h-fit">
            <Link to="/">{translate("Overview")}</Link>
          </Button>
          {auth.loginRequired && (
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
          )}
        </div>

        <div className="bg-white dark:bg-gray-800 rounded-lg shadow">
          <div className="px-6 pt-6">
            <Tabs defaultValue="users" className="w-full">
              <TabsList>
                <TabsTrigger value="users" asChild>
                  <Link to="/admin">{translate("Users")}</Link>
                </TabsTrigger>
                <TabsTrigger value="keys" asChild>
                  <Link to="/admin/keys">{translate("API Keys")}</Link>
                </TabsTrigger>
                <TabsTrigger value="security" asChild>
                  <Link to="/admin/security">{translate("Security")}</Link>
                </TabsTrigger>
              </TabsList>
            </Tabs>
          </div>

          <div className="p-6">
            <Outlet />
          </div>
        </div>
      </div>
    </div>
  );
}
