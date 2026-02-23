// SPDX-License-Identifier: MPL-2.0

import React, { useState, useEffect, useCallback } from "react";
import { cx, parseJwt } from "../../lib/utils";
import {
  RiMenuLine,
  RiLayoutLine,
  RiFileAddLine,
  RiAdminLine,
  RiLogoutBoxRLine,
  RiBook2Line,
  RiExternalLinkLine,
} from "@remixicon/react";
import { logout, getJwt } from "../../lib/auth";
import { isRedirect, Link, useNavigate, useLocation } from "@tanstack/react-router";
import { Button } from "../../components/tremor/Button";
import { MenuContext } from "../../contexts/MenuContext";
import { getSystemConfig } from "../../lib/system";
import { Tooltip } from "../tremor/Tooltip";

const isLg = () => window.innerWidth >= 1024;
const MENU_STATE_KEY = "shaper-menu-open";

export function MenuProvider ({
  children,
  isHome = false,
  isAdmin = false,
  isNewPage = false,
  currentPath = "/",
  appType,
}: {
  children: React.ReactNode;
  isHome?: boolean;
  isAdmin?: boolean;
  isNewPage?: boolean;
  currentPath?: string;
  appType?: "dashboard" | "task";
}) {
  const navigate = useNavigate();
  const location = useLocation();

  // Check if we're in dev mode from search params (reactive to location changes)
  // Use window.location.search to reliably get the current URL's search params
  const searchParams = new URLSearchParams(window.location.search);
  const isDev = searchParams.has("dev");

  // Get initial state from localStorage or default
  const getStoredMenuState = (): boolean | null => {
    const stored = localStorage.getItem(MENU_STATE_KEY);
    if (stored !== null) {
      return stored === "true";
    }
    return null; // Will use defaultOpen
  };

  const [isMenuOpen, setIsMenuOpen] = useState<boolean | null>(() => {
    if (isDev) {
      return false; // Always closed in dev mode
    }
    return getStoredMenuState();
  });
  const [defaultOpen, setDefaultOpen] = useState(isLg());
  const [extraContent, setExtraContent] = useState<React.ReactNode | null>(null);
  const [title, setTitle] = useState<string | React.ReactNode | undefined>(undefined);
  const [userName, setUserName] = useState<string>("");

  const fetchUserName = useCallback(async () => {
    try {
      const jwt = await getJwt();
      const decoded = parseJwt(jwt);
      setUserName(decoded.userName || "");
    } catch (error) {
      if (isRedirect(error)) {
        navigate(error.options);
        return;
      }
      console.error("Failed to fetch username:", error);
    }
  }, [navigate]);

  useEffect(() => {
    fetchUserName();
  }, [fetchUserName]);

  useEffect(() => {
    const handleResize = () => {
      const state = isLg();
      setDefaultOpen(state);
    };
    handleResize();
    window.addEventListener("resize", handleResize);
    return () => {
      window.removeEventListener("resize", handleResize);
    };
  }, []);

  // Persist menu state to localStorage when it changes (but not in dev mode)
  useEffect(() => {
    if (isMenuOpen !== null && !isDev) {
      localStorage.setItem(MENU_STATE_KEY, String(isMenuOpen));
    }
  }, [isMenuOpen, isDev]);

  // Update menu state when dev mode changes
  useEffect(() => {
    if (isDev) {
      // In dev mode, force closed
      setIsMenuOpen(false);
    } else {
      // When not in dev mode, restore from localStorage or use default
      const stored = localStorage.getItem(MENU_STATE_KEY);
      if (stored !== null) {
        setIsMenuOpen(stored === "true");
      } else {
        // Use null to fall back to defaultOpen (based on screen size)
        setIsMenuOpen(null);
      }
    }
  }, [isDev]);

  const actuallyOpen = isMenuOpen === null ? defaultOpen : isMenuOpen;

  // Determine the correct documentation URL based on current route
  const getDocumentationUrl = () => {
    const docsLink = "https://taleshape.com/shaper/docs";
    const pathname = location.pathname;
    // Dashboard-related routes (create, view, edit)
    if (pathname.startsWith("/dashboards/") ||
      pathname.startsWith("/dashboards_/")) {
      return docsLink + "/dashboard-sql-reference/";
    }
    // Task-related routes (create, edit)
    if (pathname.startsWith("/tasks/")) {
      return docsLink + "/tasks-and-scheduling/";
    }
    // New page - check appType to determine which docs to show
    if (pathname === "/new") {
      if (appType === "task") {
        return docsLink + "/tasks-and-scheduling/";
      } else {
        return docsLink + "/dashboard-sql-reference/";
      }
    }
    // Default documentation URL
    return docsLink;
  };

  return (
    <MenuContext.Provider value={{
      isMenuOpen: actuallyOpen,
      setIsMenuOpen,
      setExtraContent,
      setTitle,
    }}>
      <div
        className={cx(
          "fixed top-0 bottom-0 left-0 z-50 overflow-y-auto shadow-sm shadow-cb dark:shadow-db bg-cbg dark:bg-dbg w-full sm:w-56 flex flex-col",
          "print:hidden",
          {
            "hidden": !actuallyOpen,
          },
        )}
      >
        <div>
          <button
            onClick={() => {
              setIsMenuOpen(false);
            }}
          >
            <RiMenuLine className="pl-1 py-1 ml-[0.4rem] mt-[0.675rem] mb-3 size-7 text-ctext2 dark:text-dtext2 hover:text-ctext hover:dark:text-dtext transition-colors" />
          </button>
          {title && (
            <span
              className="block mx-4 font-display text-lg mb-4"
            >{title}</span>
          )}
          <Link
            to="/"
            search={isHome ? undefined : { path: currentPath }}
            className={cx("block px-4 py-3", {
              "hover:underline": !isHome,
              "bg-cprimary dark:bg-dprimary text-ctextb dark:text-dtextb": isHome,
            })}
          >
            <Tooltip content={"Go to " + (isHome ? "/" : currentPath)} showArrow={false}>
              <RiLayoutLine className="size-4 inline mr-1.5 mb-1" />
              Browse
            </Tooltip>
          </Link>
          <Link
            to="/new"
            search={{ path: currentPath }}
            disabled={isNewPage}
            className={cx("block px-4 py-3", {
              "hover:underline": !isNewPage,
              "bg-cprimary dark:bg-dprimary text-ctextb dark:text-dtextb": isNewPage,
            })}
          >
            <RiFileAddLine className="size-4 inline mr-1.5 mb-1" />
            New
          </Link>
          {extraContent}
        </div>

        <div className="mt-auto pt-4 pb-4 space-y-2">
          <a
            href={getDocumentationUrl()}
            className="block px-4 pt-2 hover:text-ctext hover:dark:text-dtext text-sm text-ctext2 dark:text-dtext2 group hover:underline"
            target="shaper-docs"
          >
            <RiBook2Line className="size-4 inline mr-1.5 mb-1" />
            Docs
            <RiExternalLinkLine className="size-3.5 inline ml-1 -mt-1 fill-ctext2 dark:fill-dtext2 opacity-0 group-hover:opacity-100 transition-opacity" />
          </a>
          <Link
            to="/admin"
            disabled={isAdmin}
            className={cx(
              "block px-4 pt-2 hover:text-ctext hover:dark:text-dtext text-sm hover:underline",
              {
                "text-ctext2 dark:text-dtext2": !isAdmin,
                "underline cursor-default": isAdmin,
              },
            )}
          >
            <RiAdminLine className="size-4 inline mr-1 -mt-1" />
            Admin
          </Link>
          {getSystemConfig().loginRequired && (
            <div className="flex items-center gap-2 pt-4 mx-4 border-t border-cb dark:border-db">
              <span className="text-sm text-ctext2 dark:text-dtext2 overflow-hidden whitespace-nowrap text-ellipsis flex-grow">
                {userName}
              </span>
              <Button
                onClick={async () => {
                  navigate((await logout()).options);
                }}
                variant="light"
              >
                <RiLogoutBoxRLine className="size-4 inline mr-0.5 -ml-0.5 -mt-0.5" />
                Logout
              </Button>
            </div>
          )}
        </div>
      </div>

      <div className={cx("@container", { "sm:ml-56 print:ml-0": actuallyOpen })}>
        {children}
      </div>
    </MenuContext.Provider>
  );
}
