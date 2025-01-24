import React, { useState, useEffect, useRef } from "react";
import { cx } from "../lib/utils";
import {
  RiMenuLine,
  RiHomeLine,
  RiFileAddLine,
  RiAdminLine,
  RiLogoutBoxRLine,
} from "@remixicon/react";
import { useAuth, logout, parseJwt } from "../lib/auth";
import { Link, useNavigate } from "@tanstack/react-router";
import { translate } from "../lib/translate";
import { Button } from "../components/tremor/Button";

export function Menu({
  children,
  inline = false,
  hideHome = false,
  hideAdmin = false,
  isNewPage = false,
  onOpenChange,
}: {
  children?: React.ReactNode;
  inline?: boolean;
  hideHome?: boolean;
  hideAdmin?: boolean;
  isNewPage?: boolean;
  onOpenChange?: (open: boolean) => void;
}) {
  const auth = useAuth();
  const navigate = useNavigate();
  const [isMenuOpen, setIsMenuOpen] = useState<boolean | null>(null);
  const [defaultOpenState, setDefaultOpenState] = useState(false);
  const [userName, setUserName] = useState<string>("");
  const menuRef = useRef<HTMLDivElement>(null);
  const actuallyOpen = isMenuOpen || (isMenuOpen === null && defaultOpenState);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        !inline &&
        menuRef.current &&
        !menuRef.current.contains(event.target as Node) &&
        isMenuOpen
      ) {
        setIsMenuOpen(false);
        if (onOpenChange) {
          onOpenChange(false);
        }
      }
    };

    document.addEventListener("mousedown", handleClickOutside);
    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, [inline, isMenuOpen, onOpenChange]);

  useEffect(() => {
    const fetchUserName = async () => {
      try {
        const jwt = await auth.getJwt();
        const decoded = parseJwt(jwt);
        setUserName(decoded.userName || "");
      } catch (error) {
        console.error("Failed to fetch username:", error);
      }
    };

    fetchUserName();
  }, [auth]);

  useEffect(() => {
    const handleResize = () => {
      // 640px is the 'sm' breakpoint
      const state = inline && window.innerWidth >= 640;
      setDefaultOpenState(state);
      if (onOpenChange && isMenuOpen === null) {
        onOpenChange(state);
      }
    };

    // Set initial state
    handleResize();

    // Add resize listener
    window.addEventListener("resize", handleResize);

    return () => {
      window.removeEventListener("resize", handleResize);
    };
  }, [inline, isMenuOpen, onOpenChange]);

  return (
    <>
      <button
        className={cx("px-1", {
          hidden: inline && actuallyOpen,
          "mr-1": inline,
        })}
        onClick={() => {
          setIsMenuOpen(true);
          if (onOpenChange) {
            onOpenChange(true);
          }
        }}
      >
        <RiMenuLine className="py-1 size-7 text-ctext2 dark:text-dtext2 hover:text-ctext hover:dark:text-dtext transition-colors" />
      </button>

      <div
        ref={menuRef}
        className={cx(
          "fixed top-0 left-0 w-full sm:w-72 h-dvh ease-in-out delay-75 duration-300 z-40 bg-cbg dark:bg-dbg",
          {
            hidden: inline && !actuallyOpen,
            "border-r border-cborder dark:border-dborder":
              inline && actuallyOpen,
            "bg-cbga dark:bg-dbga shadow-xl !ml-0": !inline,
            "-translate-x-[calc(100vw+50px)]": !inline && !actuallyOpen,
          },
        )}
      >
        <div className="flex flex-col h-full">
          <div>
            <button
              onClick={() => {
                setIsMenuOpen(false);
                if (onOpenChange) {
                  onOpenChange(false);
                }
              }}
            >
              <RiMenuLine className="pl-1 py-1 ml-2 mt-5 mb-4 size-7 text-ctext2 dark:text-dtext2 hover:text-ctext hover:dark:text-dtext transition-colors" />
            </button>
            {!hideHome && (
              <Link
                to="/"
                className="block px-4 py-4 hover:bg-cprimary dark:hover:bg-dprimary hover:text-cbga dark:hover:text-cbga transition-colors"
              >
                <RiHomeLine className="size-4 inline mr-2 mb-1" />
                {translate("Home")}
              </Link>
            )}
            {!isNewPage && (
              <Link
                to="/dashboard/new"
                className="block px-4 py-4 hover:bg-cprimary dark:hover:bg-dprimary hover:text-cbga dark:hover:text-cbga transition-colors"
              >
                <RiFileAddLine className="size-4 inline mr-2 mb-1" />
                {translate("New Dashboard")}
              </Link>
            )}
            {children}
          </div>

          <div className="mt-auto px-5 pb-6 space-y-3">
            {!hideAdmin && (
              <Link
                to="/admin"
                className="block text-ctext2 dark:text-dtext2 hover:text-ctext hover:dark:text-dtext"
              >
                <RiAdminLine className="size-4 inline mr-2 mb-1" />
                {translate("Admin")}
              </Link>
            )}
            {auth.loginRequired && (
              <div className="flex items-center gap-2 pt-2">
                <span className="text-sm text-ctext2 dark:text-dtext2 overflow-hidden whitespace-nowrap text-ellipsis flex-grow">
                  {userName}
                </span>
                <Button
                  onClick={async () => {
                    navigate(await logout());
                  }}
                  variant="secondary"
                >
                  <RiLogoutBoxRLine className="size-4 inline mr-2 mb-1" />
                  {translate("Logout")}
                </Button>
              </div>
            )}
          </div>
        </div>
      </div>
    </>
  );
}
