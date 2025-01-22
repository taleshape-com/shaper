import React, { useState, useEffect, useRef } from "react";
import { cx } from "../lib/utils";
import {
  RiMenuLine,
  RiCloseLargeLine,
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
}: {
  children?: React.ReactNode;
  inline?: boolean;
  hideHome?: boolean;
  hideAdmin?: boolean;
  isNewPage?: boolean;
}) {
  const auth = useAuth();
  const navigate = useNavigate();
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const [userName, setUserName] = useState<string>("");
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        !inline &&
        menuRef.current &&
        !menuRef.current.contains(event.target as Node) &&
        isMenuOpen
      ) {
        setIsMenuOpen(false);
      }
    };

    document.addEventListener("mousedown", handleClickOutside);
    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, [inline, isMenuOpen]);

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
      setIsMenuOpen(inline && window.innerWidth >= 640);
    };

    // Set initial state
    handleResize();

    // Add resize listener
    window.addEventListener("resize", handleResize);

    return () => {
      window.removeEventListener("resize", handleResize);
    };
  }, [inline]);

  return (
    <div>
      <button
        className={cx("px-1", { hidden: inline && isMenuOpen, "ml-4 mt-4": inline })}
        onClick={() => setIsMenuOpen(true)}
      >
        <RiMenuLine className="py-1 size-7 text-ctext2 dark:text-dtext2 hover:text-ctext hover:dark:text-dtext transition-colors" />
      </button>
      <div
        ref={menuRef}
        className={cx(
          "w-full sm:w-fit h-dvh ease-in-out delay-75 duration-300 z-40 min-w-72",
          inline
            ? cx({
              hidden: !isMenuOpen,
              "fixed sm:relative border-r border-cborder dark:border-dborder":
                isMenuOpen,
            })
            : cx("bg-cbga dark:bg-dbga fixed top-0 left-0 shadow-xl !ml-0", {
              "-translate-x-[calc(100vw+50px)]": !isMenuOpen,
            }),
        )}
      >
        <div className="flex flex-col h-full">
          <div>
            <button onClick={() => setIsMenuOpen(false)}>
              <RiCloseLargeLine className="pl-1 py-1 ml-2 mt-2 mb-4 size-7 text-ctext2 dark:text-dtext2 hover:text-ctext hover:dark:text-dtext transition-colors" />
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

          {auth.loginRequired && (
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
              <div className="flex items-center gap-2">
                <span className="text-sm text-ctext2 dark:text-dtext2 max-w-36 overflow-hidden whitespace-nowrap text-ellipsis">
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
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
