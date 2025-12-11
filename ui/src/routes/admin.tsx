// SPDX-License-Identifier: MPL-2.0

import { createFileRoute, Link, Outlet, useLocation } from "@tanstack/react-router";
import { Helmet } from "react-helmet";
import { Tabs, TabsList, TabsTrigger } from "../components/tremor/Tabs";
import { MenuProvider } from "../components/providers/MenuProvider";
import { MenuTrigger } from "../components/MenuTrigger";
import { RiAdminLine } from "@remixicon/react";
import { useQueryApi } from "../hooks/useQueryApi";
import { useEffect, useState } from "react";

export const Route = createFileRoute("/admin")({
  component: Admin,
});

function Admin () {
  const location = useLocation();
  const queryApi = useQueryApi();
  const [version, setVersion] = useState<string | null>(null);

  let selectedTab = "users";
  if (location.pathname.endsWith("/admin/keys")) {
    selectedTab = "keys";
  } else if (location.pathname.endsWith("/admin/security")) {
    selectedTab = "security";
  }

  useEffect(() => {
    const fetchVersion = async () => {
      try {
        const data = await queryApi("version") as { version: string };
        setVersion(data.version);
      } catch (error) {
        // Silently fail - version is not critical
        console.error("Failed to fetch version:", error);
      }
    };
    fetchVersion();
  }, [queryApi]);

  return (
    <MenuProvider isAdmin>
      <Helmet>
        <title>Admin</title>
        <meta name="description" content="Admin Settings" />
      </Helmet>

      <div className="px-4 pb-2 min-h-dvh flex flex-col">
        <div className="flex">
          <MenuTrigger className="pr-1.5 py-3 -ml-1.5" />
          <h1 className="text-2xl font-semibold font-display flex-grow pb-2 pt-2.5">
            <RiAdminLine className="size-5 inline mr-1 -mt-1" />
            Admin
          </h1>
        </div>

        <div className="bg-cbgs dark:bg-dbgs rounded-md shadow flex-grow flex flex-col">
          <div className="px-6 pt-6">
            <Tabs value={selectedTab} className="w-full">
              <TabsList>
                <TabsTrigger value="users" asChild>
                  <Link to="/admin">Users</Link>
                </TabsTrigger>
                <TabsTrigger value="keys" asChild>
                  <Link to="/admin/keys">API Keys</Link>
                </TabsTrigger>
                <TabsTrigger value="security" asChild>
                  <Link to="/admin/security">Security</Link>
                </TabsTrigger>
              </TabsList>
            </Tabs>
          </div>

          <div className="p-6 flex-grow">
            <Outlet />
          </div>
        </div>

        {version && (
          <div className="text-right mt-3 text-sm text-ctext/60 dark:text-dtext/60">
            <a
              href={version === "dev"
                ? "https://github.com/taleshape-com/shaper"
                : `https://github.com/taleshape-com/shaper/releases/tag/v${version}`
              }
              className="hover:underline"
              target="shaper-version"
            >Shaper {version === "dev" ? "dev" : `v${version}`}</a>
          </div>
        )}
      </div>
    </MenuProvider>
  );
}
