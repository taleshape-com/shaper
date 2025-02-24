import { createFileRoute, Link, Outlet, useLocation } from "@tanstack/react-router";
import { Helmet } from "react-helmet";
import { translate } from "../lib/translate";
import { Tabs, TabsList, TabsTrigger } from "../components/tremor/Tabs";
import { Menu } from "../components/Menu";
import { cx } from "../lib/utils";
import { useState } from "react";
import { RiAdminLine } from "@remixicon/react";

export const Route = createFileRoute("/admin")({
  component: Admin,
});

function Admin() {
  const [isInlineMenuOpen, setIsInlineMenuOpen] = useState(false);
  const location = useLocation()

  let selectedTab = "users";
  if (location.pathname === "/admin/keys") {
    selectedTab = "keys";
  } else if (location.pathname === "/admin/security") {
    selectedTab = "security";
  }

  return (
    <div className={cx("flex-1 p-4 overflow-auto", { "ml-64": isInlineMenuOpen })}>
      <Helmet>
        <title>{translate("Admin")}</title>
        <meta name="description" content="Admin Settings" />
      </Helmet>
      <div className={cx("mb-4 flex", { "-ml-2": !isInlineMenuOpen })}>
        <Menu inline isAdmin onOpenChange={setIsInlineMenuOpen} />
        <h1 className="text-2xl font-semibold font-display flex-grow">
          <RiAdminLine className="size-5 inline mr-1 -mt-1" />
          {translate("Admin")}
        </h1>
      </div>

      <div className="bg-cbgl dark:bg-dbgl rounded-lg shadow">
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

        <div className="p-6 min-h-[calc(82.95vh)]">
          <Outlet />
        </div>
      </div>
    </div>
  );
}
