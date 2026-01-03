// SPDX-License-Identifier: MPL-2.0

import { redirect } from "@tanstack/react-router";
import { z } from "zod";
import clsx, { type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cx (...args: ClassValue[]) {
  return twMerge(clsx(...args));
}

export const focusInput = [
  // base
  "focus:ring-2",
  // ring color
  "focus:ring-cprimary focus:dark:ring-dprimary",
  // border color
  "focus:border-cprimary focus:dark:border-dprimary",
];

export const focusRing = [
  // base
  "outline outline-offset-0 outline-0 focus-visible:outline-2",
  // outline color
  "outline-cprimary dark:outline-dprimary",
];

export const hasErrorInput = [
  // base
  "ring-0 focus-visible:ring-2 focus-visible:outline-0",
  // border color
  "border-red-500 dark:border-red-700",
  // ring color
  "ring-red-200 dark:ring-red-700/30",
];

export const varsParamSchema = z
  .record(z.union([z.string(), z.array(z.string())]))
  .optional();
export type VarsParamSchema = (typeof varsParamSchema)["_type"];

export const getSearchParamString = (vars: VarsParamSchema) => {
  const params = new URLSearchParams();
  Object.entries(vars ?? {}).forEach(([key, value]) => {
    if (Array.isArray(value)) {
      // To allow clearing an array param, we need to explicitly set it to empty string
      if (value.length === 0) {
        params.set(key, "");
        return;
      }
      value.forEach((v) => {
        params.append(key, v);
      });
      return;
    }
    params.set(key, value);
  });
  return params.toString();
};

export const goToLoginPage = () => {
  return redirect({
    to: "/login",
    replace: true,
    search: {
      // Use the current location to power a redirect after login
      // (Do not use `router.state.resolvedLocation` as it can
      // potentially lag behind the actual current location)
      redirect: location.pathname + location.search + location.hash,
    },
  });
};

const castRegex = /^CAST\(.+ AS .+\)$/;
const boxplotRegex = /^boxplot\(.+\)$/;
const singleQuoteEscapedRegex = /^'(?:[^']|'')*'$/;
export const getNameIfSet = (name: string) => {
  if (castRegex.test(name) || singleQuoteEscapedRegex.test(name) || boxplotRegex.test(name)) {
    return undefined;
  }
  return name;
};

export const isMac = () => navigator.userAgent.includes("Mac");

export function parseJwt (token: string) {
  const base64Url = token.split(".")[1];
  const base64 = base64Url.replace(/-/g, "+").replace(/_/g, "/");
  const jsonPayload = decodeURIComponent(
    window
      .atob(base64)
      .split("")
      .map(function (c) {
        return "%" + ("00" + c.charCodeAt(0).toString(16)).slice(-2);
      })
      .join(""),
  );
  return JSON.parse(jsonPayload);
}

export function removeTrailingSlash (s: string) {
  return s.replace(/\/+$/, "");
}

export async function copyToClipboard (text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch (err) {
    console.error("Failed to copy to clipboard, trying fallback:", err);
    // Fallback for older browsers
    try {
      const textArea = document.createElement("textarea");
      textArea.value = text;
      textArea.style.position = "fixed";
      textArea.style.opacity = "0";
      document.body.appendChild(textArea);
      textArea.select();
      document.execCommand("copy");
      document.body.removeChild(textArea);
      return true;
    } catch (fallbackErr) {
      console.error("Failed to copy to clipboard:", fallbackErr);
      return false;
    }
  }
}

export function getLocalDate (input: string | number) {
  const date = new Date(input);
  const utcDate = new Date(date.toLocaleString("en-US", { timeZone: "UTC" }));
  const localDate = new Date(date.toLocaleString("en-US"));
  const offset = utcDate.getTime() - localDate.getTime();
  date.setTime(date.getTime() + offset);
  return date;
}

export function getUTCDate (input: Date) {
  const date = new Date(input);
  const utcDate = new Date(date.toLocaleString("en-US", { timeZone: "UTC" }));
  const localDate = new Date(date.toLocaleString("en-US"));
  const offset = utcDate.getTime() - localDate.getTime();
  date.setTime(date.getTime() - offset);
  return date;
}

// Helper to determine current render mode from global shaper config.
// Defaults to "interactive" when not set.
export const getRenderMode = (): "interactive" | "pdf" => {
  if (typeof window === "undefined") {
    return "interactive";
  }
  // eslint-disable-next-line @typescript-eslint/ban-ts-comment
  // @ts-ignore
  const mode = window.shaper?.renderMode;
  return mode === "pdf" ? "pdf" : "interactive";
};
