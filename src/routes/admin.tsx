import { createFileRoute, Link, Outlet } from "@tanstack/react-router";
import { Helmet } from "react-helmet";
import { translate } from "../lib/translate";
import { Tabs, TabsList, TabsTrigger } from "../components/tremor/Tabs";
import { Menu } from "../components/Menu";
import { cx } from "../lib/utils";
import { useState } from "react";

export const Route = createFileRoute("/admin")({
  component: Admin,
});

function Admin() {
  const [isMenuOpen, setIsMenuOpen] = useState(false);

  return (
    <div className={cx("flex-1 p-4 overflow-auto", { "ml-72": isMenuOpen })}>
      <Helmet>
        <title>{translate("Admin")}</title>
        <meta name="description" content="Admin Settings" />
      </Helmet>
      <div className={cx("mb-4 flex", { "-ml-2": !isMenuOpen })}>
        <Menu inline hideAdmin onOpenChange={setIsMenuOpen} />
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
  );
}
