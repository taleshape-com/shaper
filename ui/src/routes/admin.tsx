import { createFileRoute, Link, Outlet, useLocation } from "@tanstack/react-router";
import { Helmet } from "react-helmet";
import { translate } from "../lib/translate";
import { Tabs, TabsList, TabsTrigger } from "../components/tremor/Tabs";
import { MenuProvider } from "../components/providers/MenuProvider";
import { MenuTrigger } from "../components/MenuTrigger";
import { RiAdminLine } from "@remixicon/react";

export const Route = createFileRoute("/admin")({
  component: Admin,
});

function Admin() {
  const location = useLocation()

  let selectedTab = "users";
  if (location.pathname.endsWith("/admin/keys")) {
    selectedTab = "keys";
  } else if (location.pathname.endsWith("/admin/security")) {
    selectedTab = "security";
  }

  return (
    <MenuProvider isAdmin>
      <Helmet>
        <title>{translate("Admin")}</title>
        <meta name="description" content="Admin Settings" />
      </Helmet>

      <div className="px-4 pb-4 min-h-dvh flex flex-col">
        <div className="flex">
          <MenuTrigger className="pr-1.5 py-3 -ml-1.5" />
          <h1 className="text-2xl font-semibold font-display flex-grow pb-2 pt-2.5">
            <RiAdminLine className="size-5 inline mr-1 -mt-1" />
            {translate("Admin")}
          </h1>
        </div>

        <div className="bg-cbgs dark:bg-dbgs rounded-lg shadow flex-grow">
          <div className="px-6 pt-6">
            <Tabs value={selectedTab} className="w-full">
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
    </MenuProvider>
  );
}
