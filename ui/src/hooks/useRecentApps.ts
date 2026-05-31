import { useState, useEffect, useCallback } from "react";

export interface RecentApp {
  id: string;
  name: string;
  type?: string;
  timestamp: number;
}

const STORAGE_KEY = "shaper-recent-apps";
const MAX_RECENT = 10;

export function useRecentApps () {
  const [recentApps, setRecentApps] = useState<RecentApp[]>([]);

  // Load from localStorage on mount
  useEffect(() => {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored) {
      try {
        const parsed = JSON.parse(stored);
        setRecentApps(Array.isArray(parsed) ? parsed : []);
      } catch {
        setRecentApps([]);
      }
    }
  }, []);

  // Listen for changes from other components
  useEffect(() => {
    const handleStorageChange = (e: StorageEvent) => {
      if (e.key === STORAGE_KEY && e.newValue) {
        try {
          const parsed = JSON.parse(e.newValue);
          setRecentApps(Array.isArray(parsed) ? parsed : []);
        } catch {
          setRecentApps([]);
        }
      }
    };

    // Listen for storage events (from other tabs/windows)
    window.addEventListener("storage", handleStorageChange);

    // Listen for custom events (from same page)
    const handleCustomEvent = (e: Event) => {
      const customEvent = e as CustomEvent<string>;
      try {
        const parsed = JSON.parse(customEvent.detail);
        setRecentApps(Array.isArray(parsed) ? parsed : []);
      } catch {
        setRecentApps([]);
      }
    };

    window.addEventListener("recent-apps-updated", handleCustomEvent);

    return () => {
      window.removeEventListener("storage", handleStorageChange);
      window.removeEventListener("recent-apps-updated", handleCustomEvent);
    };
  }, []);

  const addRecentApp = useCallback((id: string, name: string, type?: string) => {
    setRecentApps((currentApps) => {
      const timestamp = Date.now();
      const newApp: RecentApp = { id, name, type, timestamp };

      const filtered = currentApps.filter((app) => app.id !== id);
      const updated = [newApp, ...filtered].slice(0, MAX_RECENT);

      const updatedJson = JSON.stringify(updated);
      localStorage.setItem(STORAGE_KEY, updatedJson);

      // Dispatch custom event to notify other components on the same page
      window.dispatchEvent(
        new CustomEvent("recent-apps-updated", { detail: updatedJson }),
      );

      return updated;
    });
  }, []);

  const getRecentApps = useCallback((): RecentApp[] => {
    return recentApps.slice(0, MAX_RECENT);
  }, [recentApps]);

  return { addRecentApp, getRecentApps };
}
