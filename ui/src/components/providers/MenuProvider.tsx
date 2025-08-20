// SPDX-License-Identifier: MPL-2.0

import React, { useState, useEffect, useCallback } from "react";
import { cx, parseJwt } from "../../lib/utils";
import {
  RiMenuLine,
  RiHomeLine,
  RiFileAddLine,
  RiAdminLine,
  RiLogoutBoxRLine,
  RiBook2Line,
  RiExternalLinkLine,
} from "@remixicon/react";
import { logout, getJwt } from "../../lib/auth";
import { isRedirect, Link, useNavigate } from "@tanstack/react-router";
import { translate } from "../../lib/translate";
import { Button } from "../../components/tremor/Button";
import { MenuContext } from "../../contexts/MenuContext";
import { getSystemConfig } from "../../lib/system";

const isLg = () => window.innerWidth >= 1024;

export function MenuProvider({
  children,
  isHome = false,
  isAdmin = false,
  isNewPage = false,
}: {
  children: React.ReactNode;
  isHome?: boolean;
  isAdmin?: boolean;
  isNewPage?: boolean;
}) {
  const navigate = useNavigate();
  const [isMenuOpen, setIsMenuOpen] = useState<boolean | null>(null);
  const [defaultOpen, setDefaultOpen] = useState(isLg())
  const [extraContent, setExtraContent] = useState<React.ReactNode | null>(null);
  const [title, setTitle] = useState<string | undefined>(undefined);
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

  const actuallyOpen = isMenuOpen === null ? defaultOpen : isMenuOpen;

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
            disabled={isHome}
            className={cx("block px-4 py-3", {
              "hover:underline": !isHome,
              "bg-cprimary dark:bg-dprimary text-ctexti dark:text-dtexti": isHome,
            })}
          >
            <RiHomeLine className="size-4 inline mr-1.5 mb-1" />
            {translate("Home")}
          </Link>
          <Link
            to="/new"
            disabled={isNewPage}
            className={cx("block px-4 py-3", {
              "hover:underline": !isNewPage,
              "bg-cprimary dark:bg-dprimary text-ctexti dark:text-dtexti": isNewPage,
            })}
          >
            <RiFileAddLine className="size-4 inline mr-1.5 mb-1" />
            {translate("New")}
          </Link>
          {extraContent}
        </div>

        <div className="mt-auto pt-4 pb-4 space-y-2">
          <a
            href="https://taleshape.com/shaper/docs"
            className="block px-4 pt-2 hover:text-ctext hover:dark:text-dtext text-sm text-ctext2 dark:text-dtext2 group hover:underline"
            target="shaper-docs"
          >
            <RiBook2Line className="size-4 inline mr-1.5 mb-1" />
            {translate("Docs")}
            <RiExternalLinkLine className="size-3.5 inline ml-1 -mt-1 fill-ctext2 dark:fill-dtext2 opacity-0 group-hover:opacity-100 transition-opacity" />
          </a>
          <Link to="/admin" disabled={isAdmin} className={cx(
            "block px-4 pt-2 hover:text-ctext hover:dark:text-dtext text-sm hover:underline",
            { "text-ctext2 dark:text-dtext2": !isAdmin }
          )}>
            <RiAdminLine className="size-4 inline mr-1 -mt-1" />
            {translate("Admin")}
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
                {translate("Logout")}
              </Button>
            </div>
          )}
        </div>
      </div>

      <div className={cx({ "sm:ml-56 overflow-auto": actuallyOpen })}>
        {children}
      </div>
    </MenuContext.Provider>
  );
}
