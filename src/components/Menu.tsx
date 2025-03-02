import React, { useState, useEffect, useRef, useCallback } from "react";
import { cx } from "../lib/utils";
import {
  RiMenuLine,
  RiHomeLine,
  RiFileAddLine,
  RiAdminLine,
  RiLogoutBoxRLine,
} from "@remixicon/react";
import { useAuth, logout, parseJwt } from "../lib/auth";
import { isRedirect, Link, useNavigate } from "@tanstack/react-router";
import { translate } from "../lib/translate";
import { Button } from "../components/tremor/Button";

const isLg = () => window.innerWidth >= 1024;

export function Menu({
  children,
  inline = false,
  isHome = false,
  isAdmin = false,
  isNewPage = false,
  onOpenChange,
}: {
  children?: React.ReactNode;
  inline?: boolean;
  isHome?: boolean;
  isAdmin?: boolean;
  isNewPage?: boolean;
  onOpenChange?: (open: boolean) => void;
}) {
  const { getJwt, loginRequired } = useAuth();
  const navigate = useNavigate();
  const [isMenuOpen, setIsMenuOpen] = useState<boolean | null>(null);
  const [actuallyInline, setActuallyInline] = useState(inline && isLg());
  const [userName, setUserName] = useState<string>("");
  const menuRef = useRef<HTMLDivElement>(null);
  const actuallyOpen = isMenuOpen || (isMenuOpen === null && actuallyInline);

  const fetchUserName = useCallback(async () => {
    try {
      const jwt = await getJwt();
      const decoded = parseJwt(jwt);
      setUserName(decoded.userName || "");
    } catch (error) {
      if (isRedirect(error)) {
        navigate(error);
        return;
      }
      console.error("Failed to fetch username:", error);
    }
  }, [getJwt, navigate]);


  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        !actuallyInline &&
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
  }, [actuallyInline, isMenuOpen, onOpenChange]);

  useEffect(() => {
    fetchUserName();
  }, [fetchUserName]);

  useEffect(() => {
    const handleResize = () => {
      const state = inline && isLg();
      setActuallyInline(state);
      if (onOpenChange && (isMenuOpen === null || isMenuOpen)) {
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
          hidden: actuallyInline && actuallyOpen,
          "mr-1": actuallyInline,
        })}
        onClick={() => {
          setIsMenuOpen(true);
          if (onOpenChange) {
            onOpenChange(actuallyInline);
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
            hidden: actuallyInline && !actuallyOpen,
            "border-r border-cbga dark:border-dbg sm:w-64":
              actuallyInline && actuallyOpen,
            "bg-cbga dark:bg-dbga shadow-xl !ml-0": !actuallyInline,
            "-translate-x-[calc(100vw+50px)]": !actuallyInline && !actuallyOpen,
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
              <RiMenuLine className="pl-1 py-1 ml-2 mt-3 mb-3 size-7 text-ctext2 dark:text-dtext2 hover:text-ctext hover:dark:text-dtext transition-colors" />
            </button>
            <Link
              to="/"
              disabled={isHome}
              className={cx("block px-4 py-4", {
                "hover:underline": !isHome,
                "bg-cprimary dark:bg-dprimary text-cbga dark:text-cbga": isHome,
              })}
            >
              <RiHomeLine className="size-4 inline mr-2 mb-1" />
              {translate("Home")}
            </Link>
            <Link
              to="/dashboard/new"
              disabled={isNewPage}
              className={cx("block px-4 py-4", {
                "hover:underline": !isNewPage,
                "bg-cprimary dark:bg-dprimary text-cbga dark:text-cbga": isNewPage,
              })}
            >
              <RiFileAddLine className="size-4 inline mr-2 mb-1" />
              {translate("New Dashboard")}
            </Link>
            {children}
          </div>

          <div className="mt-auto pb-4 space-y-3">
            <Link to="/admin" disabled={isAdmin} className={cx(
              "block px-4 pt-2 dark:text-dtext2 hover:text-ctext hover:dark:text-dtext text-sm",
              { "text-ctext2": !isAdmin }
            )}>
              <RiAdminLine className="size-4 inline mr-1 -mt-1" />
              {translate("Admin")}
            </Link>
            {loginRequired && (
              <div className="flex items-center gap-2 pt-4 mx-4 border-t border-cb dark:border-dbga">
                <span className="text-sm text-ctext2 dark:text-dtext2 overflow-hidden whitespace-nowrap text-ellipsis flex-grow">
                  {userName}
                </span>
                <Button
                  onClick={async () => {
                    navigate(await logout());
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
      </div>
    </>
  );
}
