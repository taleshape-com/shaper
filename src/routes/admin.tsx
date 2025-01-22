import { createFileRoute, Link, Outlet } from "@tanstack/react-router";
import { Helmet } from "react-helmet";
import { translate } from "../lib/translate";
import { Tabs, TabsList, TabsTrigger } from "../components/tremor/Tabs";
import { Menu } from "../components/Menu";

export const Route = createFileRoute("/admin")({
  component: Admin,
});

function Admin() {
  return (
    <div className="flex h-screen bg-gray-50 dark:bg-gray-900">
      <Menu inline hideAdmin />
      <div className="flex-1 p-4 overflow-auto">
        <Helmet>
          <title>{translate("Admin")}</title>
          <meta name="description" content="Admin Settings" />
        </Helmet>
        <div className="mb-4 flex items-center">
          <h1 className="text-3xl font-semibold font-display flex-grow">
            {translate("Admin")}
          </h1>
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
