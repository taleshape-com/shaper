// SPDX-License-Identifier: MPL-2.0

import { useEffect, useState } from "react";
import { translate } from "../lib/translate";

interface RelativeDateProps {
  date: Date;
  refresh?: boolean;
}

export function RelativeDate({ date, refresh }: RelativeDateProps) {
  const [text, setText] = useState<string>("");

  const formatRelativeDate = (targetDate: Date) => {
    const now = new Date();
    const diffMs = targetDate.getTime() - now.getTime();
    const absMs = Math.abs(diffMs);

    const seconds = Math.round(absMs / 1000);
    const minutes = Math.round(seconds / 60);
    const hours = Math.round(minutes / 60);
    const days = Math.round(hours / 24);

    // More than 30 days in either direction - use locale string
    if (days > 30) {
      return targetDate.toLocaleString();
    }

    const isFuture = diffMs > 0;

    // Less than 2 minutes
    if (minutes < 2) {
      const key = isFuture ? "in %% seconds" : "%% seconds ago";
      return translate(key).replace("%%", seconds.toString());
    }

    // Less than 2 hours
    if (hours < 2) {
      const key = isFuture ? "in %% minutes" : "%% minutes ago";
      return translate(key).replace("%%", minutes.toString());
    }

    // Less than 2 days
    if (days < 2) {
      const key = isFuture ? "in %% hours" : "%% hours ago";
      return translate(key).replace("%%", hours.toString());
    }

    // 2 or more days
    const key = isFuture ? "in %% days" : "%% days ago";
    return translate(key).replace("%%", days.toString());
  };

  useEffect(() => {
    const updateText = () => {
      setText(formatRelativeDate(date));
    };

    // Initial update
    updateText();

    if (!refresh) {
      return
    }

    // Update every 10 seconds
    const interval = setInterval(updateText, 10000);

    return () => clearInterval(interval);
  }, [date, refresh]);

  return <span>{text}</span>;
}