import { redirect } from "@tanstack/react-router";
import { z } from "zod";
import clsx, { type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cx(...args: ClassValue[]) {
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
  const urlVars = Object.entries(vars ?? {}).reduce((acc, [key, value]) => {
    if (Array.isArray(value)) {
      return [...acc, ...value.map((v) => [key, v])];
    }
    return [...acc, [key, value]];
  }, [] as string[][]);
  return new URLSearchParams(urlVars).toString();
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
const singleQuoteEscapedRegex = /^'(?:[^']|'')*'$/;
export const getNameIfSet = (name: string) => {
  if (castRegex.test(name) || singleQuoteEscapedRegex.test(name)) {
    return undefined;
  }
  return name;
};

export const isMac = () => navigator.userAgent.includes("Mac");

export function parseJwt(token: string) {
  const base64Url = token.split(".")[1];
  const base64 = base64Url.replace(/-/g, "+").replace(/_/g, "/");
  const jsonPayload = decodeURIComponent(
    window
      .atob(base64)
      .split("")
      .map(function(c) {
        return "%" + ("00" + c.charCodeAt(0).toString(16)).slice(-2);
      })
      .join(""),
  );
  return JSON.parse(jsonPayload);
}

export function removeTrailingSlash(s: string) {
  return s.replace(/\/+$/, "");
}
