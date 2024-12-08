import { z } from 'zod'
import clsx, { type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cx(...args: ClassValue[]) {
  return twMerge(clsx(...args))
}

export const focusInput = [
  // base
  "focus:ring-2",
  // ring color
  "focus:ring-blue-200 focus:dark:ring-blue-700/30",
  // border color
  "focus:border-blue-500 focus:dark:border-blue-700",
]

export const focusRing = [
  // base
  "outline outline-offset-0 outline-0 focus-visible:outline-2",
  // outline color
  "outline-cprimary dark:outline-dprimary",
]

export const hasErrorInput = [
  // base
  "ring-0 focus-visible:ring-2 focus-visible:outline-0",
  // border color
  "border-red-500 dark:border-red-700",
  // ring color
  "ring-red-200 dark:ring-red-700/30",
]


export const varsParamSchema = z.record(z.union([z.string(), z.array(z.string())])).optional()
export type VarsParamSchema = typeof varsParamSchema['_type']

export const getSearchParamString = (vars: VarsParamSchema) => {
  const urlVars = Object.entries(vars ?? {}).reduce((acc, [key, value]) => {
    if (Array.isArray(value)) {
      return [...acc, ...value.map((v) => [key, v])]
    }
    return [...acc, [key, value]]
  }, [] as string[][])
  return new URLSearchParams(urlVars).toString()
}


