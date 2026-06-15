import React, { useState, useRef, useEffect, useContext } from "react";
import { useNavigate, useLocation } from "@tanstack/react-router";
import {
  RiTimeLine,
  RiSearchLine,
  RiBarChart2Line,
  RiCodeSSlashFill,
  RiFolderFill,
} from "@remixicon/react";
import { useRecentApps, type RecentApp } from "../hooks/useRecentApps";
import { MenuContext } from "../contexts/MenuContext";
import { cx, focusInput, isMac } from "../lib/utils";
import { Tooltip } from "./tremor/Tooltip";
import { useQueryApi } from "../hooks/useQueryApi";

interface App {
  id: string;
  name: string;
  path: string;
  type: string;
}

export function SearchBar () {
  const [query, setQuery] = useState("");
  const [isActive, setIsActive] = useState(false);
  const [searchResults, setSearchResults] = useState<App[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [selectedIndex, setSelectedIndex] = useState(-1);
  const [showEmptyState, setShowEmptyState] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const navigate = useNavigate();
  const location = useLocation();
  const { getRecentApps } = useRecentApps();
  const { setIsMenuOpen } = useContext(MenuContext);
  const queryApi = useQueryApi();

  // Extract current app ID from URL
  const currentAppId = location.pathname.match(/\/(dashboards_?|tasks)\/([^/]+)/)?.[2];

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        e.stopPropagation();

        if (!isActive) {
          setIsMenuOpen(true);
          setIsActive(true);
          setTimeout(() => {
            inputRef.current?.focus();
          }, 100);
        }
      }
    };

    document.addEventListener("keydown", handleKeyDown, true);
    return () => {
      document.removeEventListener("keydown", handleKeyDown, true);
    };
  }, [setIsMenuOpen, isActive]);

  useEffect(() => {
    if (!query) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setSearchResults([]);
      setIsLoading(false);
      setShowEmptyState(false);
      return;
    }

    const controller = new AbortController();
    let emptyStateTimeoutId: NodeJS.Timeout;

    const searchTimeoutId = setTimeout(async () => {
      setIsLoading(true);
      setShowEmptyState(false);

      try {
        const data = await queryApi(
          `apps?query=${encodeURIComponent(query)}&limit=10&recursive=true`,
        );
        const results = data.apps || [];
        setSearchResults(results);

        // Only show empty state after 200ms delay if still no results
        if (results.length === 0) {
          emptyStateTimeoutId = setTimeout(() => {
            setShowEmptyState(true);
          }, 200);
        }
      } catch (error) {
        if (error instanceof Error && error.name !== "AbortError") {
          console.error("Search error:", error);
        }
      } finally {
        setIsLoading(false);
      }
    }, 300);

    return () => {
      clearTimeout(searchTimeoutId);
      clearTimeout(emptyStateTimeoutId);
      controller.abort();
    };
  }, [query, queryApi]);

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(e.target as Node) &&
        inputRef.current &&
        !inputRef.current.contains(e.target as Node)
      ) {
        setIsActive(false);
        setQuery("");
        setSelectedIndex(-1);
      }
    };

    document.addEventListener("mousedown", handleClickOutside);
    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, []);

  const handleAppClick = (appId: string, appType: string, appPath?: string, appName?: string) => {
    if (appType === "_folder") {
      // Navigate to browse page with folder path
      const folderPath = `${appPath}${appName}/`;
      navigate({ to: "/", search: { path: folderPath } });
    } else if (appType === "task") {
      navigate({ to: `/tasks/${appId}` as any });
    } else {
      navigate({ to: `/dashboards/${appId}` as any });
    }

    setIsActive(false);
    setQuery("");
    setSelectedIndex(-1);
  };

  const recentApps = getRecentApps().filter((app) => app.id !== currentAppId);
  const showRecent = !query && recentApps.length > 0;
  const showResults = query && searchResults.length > 0;
  const showEmpty = query && !isLoading && searchResults.length === 0 && showEmptyState;
  const showDropdown = isActive && (showRecent || showResults || showEmpty || isLoading);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    // Get fresh values at the time of the keypress
    const currentRecentApps = getRecentApps().filter((app) => app.id !== currentAppId);
    const allItems = query ? searchResults : currentRecentApps;

    if (e.key === "ArrowDown") {
      e.preventDefault();
      setSelectedIndex((prev) => {
        return prev < allItems.length - 1 ? prev + 1 : prev;
      });
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setSelectedIndex((prev) => {
        const newIndex = prev > 0 ? prev - 1 : -1;
        return newIndex;
      });
    } else if (e.key === "Enter" && selectedIndex >= 0) {
      e.preventDefault();
      const item = allItems[selectedIndex];
      if (item) {
        handleAppClick(
          item.id,
          item.type || "dashboard",
          "path" in item ? item.path : undefined,
          item.name,
        );
      }
    } else if (e.key === "Escape") {
      setIsActive(false);
      setQuery("");
      setSelectedIndex(-1);
      inputRef.current?.blur();
    }
  };

  // Track previous query to detect mode switches
  const prevQueryRef = useRef(query);
  const prevIsActiveRef = useRef(isActive);

  useEffect(() => {
    // Only auto-select when dropdown opens or when switching between search/recent modes
    const justOpened = isActive && !prevIsActiveRef.current;
    const modeSwitched = isActive && (!!query !== !!prevQueryRef.current);

    if (justOpened || modeSwitched) {
      const items = query ? searchResults : recentApps;
      setSelectedIndex(items.length > 0 ? 0 : -1);
    }

    prevQueryRef.current = query;
    prevIsActiveRef.current = isActive;
  }, [query, isActive, searchResults, recentApps]);

  if (!isActive) {
    const shortcut = isMac() ? "⌘K" : "Ctrl+K";
    return (
      <button
        onClick={() => {
          setIsActive(true);
          setTimeout(() => inputRef.current?.focus(), 0);
        }}
        className={cx("block w-full px-4 py-3 text-left")}
      >
        <Tooltip content={shortcut} showArrow={false}>
          <RiSearchLine className="size-4 inline mr-1.5 mb-1" />
          <span className="hover:underline">Search</span>
        </Tooltip>
      </button>
    );
  }

  return (
    <div className="relative px-4 pb-1.5 pt-1">
      <div className="relative">
        <input
          ref={inputRef}
          type="text"
          placeholder="Search"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onKeyDown={handleKeyDown}
          autoFocus
          className={cx(
            "relative block w-full appearance-none rounded-md border px-2.5 py-[0.4375rem] pl-8 shadow-sm outline-none transition sm:text-sm",
            "caret-ctext dark:caret-dtext",
            "border-cb dark:border-db",
            "text-ctext dark:text-dtext",
            "placeholder-ctext2 dark:placeholder-dtext2",
            "bg-cbg dark:bg-dbg",
            focusInput,
          )}
        />
        <div
          className={cx(
            "pointer-events-none absolute top-0 left-2 flex h-full items-center justify-center",
            "text-gray-400 dark:text-gray-600",
          )}
        >
          <RiSearchLine className="size-[1.125rem] shrink-0" aria-hidden="true" />
        </div>
      </div>

      {showDropdown && (
        <div
          ref={dropdownRef}
          className={cx(
            "absolute left-4 right-4 z-50 mt-1 max-h-80 overflow-y-auto rounded-md border shadow-lg",
            "border-cb dark:border-db bg-cbg dark:bg-dbg",
          )}
        >
          {showRecent && (
            <div className="py-1">
              <div className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-ctext2 dark:text-dtext2">
                <RiTimeLine className="size-3.5" />
                <span>Recent</span>
              </div>
              {recentApps.map((app: RecentApp, index: number) => {
                const appType = app.type || "dashboard";
                return (
                  <button
                    key={app.id}
                    onClick={() => handleAppClick(app.id, appType)}
                    className={cx(
                      "flex items-center gap-2 w-full px-3 py-2 text-left text-sm transition-colors",
                      "text-ctext dark:text-dtext",
                      {
                        "bg-gray-100 dark:bg-gray-800": selectedIndex === index,
                        "hover:bg-gray-100 dark:hover:bg-gray-800":
                          selectedIndex !== index,
                      },
                    )}
                  >
                    {appType === "_folder" ? (
                      <RiFolderFill
                        className="size-4 shrink-0 fill-ctext2 dark:fill-dtext2"
                        aria-hidden={true}
                      />
                    ) : appType === "task" ? (
                      <RiCodeSSlashFill
                        className="size-4 shrink-0 fill-ctext2 dark:fill-dtext2"
                        aria-hidden={true}
                      />
                    ) : (
                      <RiBarChart2Line
                        className="size-4 shrink-0 fill-ctext2 dark:fill-dtext2"
                        aria-hidden={true}
                      />
                    )}
                    <span>{app.name}</span>
                  </button>
                );
              })}
            </div>
          )}

          {showResults && (
            <div className="py-1">
              {searchResults.map((app: App, index: number) => (
                <button
                  key={app.id}
                  onClick={() => handleAppClick(app.id, app.type, app.path, app.name)}
                  className={cx(
                    "flex items-start gap-2 w-full px-3 py-2 text-left text-sm transition-colors",
                    "text-ctext dark:text-dtext",
                    {
                      "bg-gray-100 dark:bg-gray-800": selectedIndex === index,
                      "hover:bg-gray-100 dark:hover:bg-gray-800":
                        selectedIndex !== index,
                    },
                  )}
                >
                  {app.type === "_folder" ? (
                    <RiFolderFill
                      className="size-4 shrink-0 mt-0.5 fill-ctext2 dark:fill-dtext2"
                      aria-hidden={true}
                    />
                  ) : app.type === "task" ? (
                    <RiCodeSSlashFill
                      className="size-4 shrink-0 mt-0.5 fill-ctext2 dark:fill-dtext2"
                      aria-hidden={true}
                    />
                  ) : (
                    <RiBarChart2Line
                      className="size-4 shrink-0 mt-0.5 fill-ctext2 dark:fill-dtext2"
                      aria-hidden={true}
                    />
                  )}
                  <div className="flex-1 min-w-0">
                    <div className="font-medium">{app.name}</div>
                    {app.path !== "/" && (
                      <div className="text-xs text-ctext2 dark:text-dtext2 truncate">
                        {app.path}
                      </div>
                    )}
                  </div>
                </button>
              ))}
            </div>
          )}

          {showEmpty && (
            <div className="px-3 py-6 text-center text-sm text-ctext2 dark:text-dtext2">
              No results
            </div>
          )}

          {isLoading && (
            <div className="px-3 py-6 text-center text-sm text-ctext2 dark:text-dtext2">
              Searching...
            </div>
          )}
        </div>
      )}
    </div>
  );
}
