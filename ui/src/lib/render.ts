// SPDX-License-Identifier: MPL-2.0

import { Column } from "./types";
import * as echarts from "echarts/core";

// We interpret the dates as local time to disaply them the same way no matter which timezone a user is in.
// Two people in the same company should be looking at the same timestamps, no matter where they are right now.
// The backend returns dates as unix timestamp integers in milliseconds, but this function works for strings as well
function parseLocalDate (d: string | number) {
  return new Date(d);
}

export function formatIntegerPart (str: string): string {
  if (str.length < 5) return str;
  return str.replace(/\B(?=(\d{3})+(?!\d))/g, "\u2009");
}

export function formatFractionPart (str: string): string {
  if (str.length < 5) return str;
  const parts: string[] = [];
  for (let i = 0; i < str.length; i += 3) {
    parts.push(str.slice(i, i + 3));
  }
  return parts.join("\u2009");
}

export const formatValue = (value: string | number | boolean | null | undefined, columnType: Column["type"], shouldFormatNumbers?: boolean, shortFormat?: boolean | number) => {
  if (value === null || value === undefined) {
    return "";
  }
  if (columnType === "array" || columnType === "object") {
    return JSON.stringify(value);
  }
  if (typeof value === "boolean") {
    return value ? "YES" : "NO";
  }
  const d = parseLocalDate(value);
  if (columnType === "year") {
    return d.getFullYear().toString();
  }
  if (columnType === "month") {
    return d.toLocaleString(navigator.languages, { year: "numeric", month: "short", timeZone: "UTC" });
  }
  if (columnType === "date") {
    return d.toLocaleString(navigator.languages, { year: "numeric", month: shortFormat ? "numeric" : "short", day: "numeric", timeZone: "UTC", weekday: shortFormat ? undefined : "short" });
  }
  if (columnType === "hour") {
    return d.toLocaleString(navigator.languages, { year: shortFormat ? undefined : "numeric", month: shortFormat ? "numeric" : "short", day: "numeric", hour: "numeric", minute: shortFormat ? undefined : "2-digit", timeZone: "UTC", weekday: shortFormat ? undefined : "short" });
  }
  if (columnType === "timestamp") {
    return d.toLocaleString(navigator.languages, { year: "numeric", month: shortFormat ? "numeric" : "short", day: "numeric", hour: shortFormat ? undefined : "numeric", minute: shortFormat ? undefined : "2-digit", second: shortFormat ? undefined : "2-digit", hourCycle: "h24", timeZone: "UTC" });
  }
  if (columnType === "duration" && !value) {
    return "0";
  }
  if (typeof value === "number") {
    if (shouldFormatNumbers && columnType === "number") {
      const rawValueString = shortFormat
        ? (Math.round(value * 100) / 100).toString()
        : value.toString();

      if (rawValueString.includes("e") || rawValueString.includes("E")) {
        return rawValueString;
      }

      const isNegative = rawValueString.startsWith("-");
      const absString = isNegative ? rawValueString.slice(1) : rawValueString;

      const parts = absString.split(".");
      const integerPart = parts[0];
      const fractionPart = parts[1];

      const formattedInteger = formatIntegerPart(integerPart);
      const formattedFraction = fractionPart !== undefined ? formatFractionPart(fractionPart) : undefined;

      let result = formattedInteger;
      if (formattedFraction !== undefined) {
        result += "." + formattedFraction;
      }
      return isNegative ? "-" + result : result;
    }
    // duration comes in ms
    if (columnType === "duration") {
      const day = Math.floor(value / 86400000);
      const hours = Math.floor((value % 86400000) / 3600000);
      const minutes = Math.floor((value % 3600000) / 60000);
      const seconds = Math.floor((value % 60000) / 1000);
      const ms = Math.floor(value % 1000);
      const mainParts = [];
      if (day > 0) {
        mainParts.push(`${day}d`);
      }
      if (!shortFormat || value < 864000000) {
        if (hours > 0) {
          mainParts.push(`${hours}h`);
        }
      }
      if (!shortFormat || value < 86400000) {
        if (minutes > 0) {
          mainParts.push(`${minutes}m`);
        }
      }
      if (!shortFormat || value < 3600000) {
        if (!shortFormat && ms > 0) {
          mainParts.push(`${seconds}.${ms.toString().padStart(3, "0")}s`);
        } else if (seconds > 0) {
          mainParts.push(`${seconds}s`);
        } else if (minutes >= 0 && hours <= 0 && day <= 0) {
          mainParts.push("0s");
        }
      }
      if (mainParts.length === 0) {
        return "0s";
      }
      return mainParts.join(" ");
    }
    if (columnType === "time") {
      const hours = Math.floor(value / 3600000);
      const minutes = Math.floor((value % 3600000) / 60000);
      const seconds = Math.floor((value % 60000) / 1000);
      const ms = Math.floor(value % 1000);
      const timeString = `${String(hours).padStart(2, "0")}:${String(minutes).padStart(2, "0")}${shortFormat ? "" : `:${String(seconds).padStart(2, "0")}`}`;
      return !shortFormat && ms > 0 ? `${timeString}.${String(ms).padStart(3, "0")}` : timeString;
    }
    if (columnType === "percent") {
      return `${Math.round(value * 10000) / 100}%`;
    }
  }
  if (typeof value === "string" && shortFormat) {
    const maxLen = typeof shortFormat === "number" ? Math.max(Math.round(shortFormat), 12) : 30;
    if (value.length > maxLen) {
      return value.slice(0, maxLen) + "…";
    }
  }
  return value;
};

export const formatCellValue = (value: string | number | boolean | null | undefined) => {
  if (value === null || value === undefined) {
    return "";
  }
  if (typeof value === "boolean") {
    return value ? 0 : 1;
  }
  return value;
};

export const isJSONType = (columnType: Column["type"]) => {
  return columnType === "array" || columnType === "object";
};

export const echartsEncode = (v: string | number) => {
  return echarts.format.encodeHTML(v.toString());
};

export function toCssId (v: string | number) {
  return "shaper-" + (v.toString()
    // Replace spaces and special characters with hyphens
    .replace(/[^a-zA-Z0-9_-]/g, "-")
    // Remove consecutive hyphens
    .replace(/-+/g, "-")
    // Remove leading/trailing hyphens
    .replace(/^-+|-+$/g, "")
    // Handle empty string case
    || Math.random().toString(36).substring(2, 10)).toLowerCase();
}
