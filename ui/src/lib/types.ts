// SPDX-License-Identifier: MPL-2.0

export interface IApp {
  id: string;
  name: string;
  path?: string;
  folderId?: string;
  content: string;
  createdAt: string;
  updatedAt: string;
  createdBy?: string;
  updatedBy?: string;
  visibility?: "public" | "private" | "password-protected";
  type: "dashboard" | "task" | "_folder";
  taskInfo?: ITaskInfo;
}

interface ITaskInfo {
  lastRunAt?: string;
  lastRunSuccess?: boolean;
  lastRunDuration?: number; // in milliseconds
  nextRunAt?: string;
}

export type IDashboard = Omit<IApp, "type">;

export type Column = {
  name: string;
  type:
  | "year"
  | "month"
  | "hour"
  | "date"
  | "timestamp"
  | "duration"
  | "time"
  | "number"
  | "string"
  | "boolean"
  | "object"
  | "array"
  | "percent";
  nullable: boolean;
  tag:
  | "index"
  | "category"
  | "value"
  | "label"
  | "hint"
  | "download"
  | "default"
  | "defaultFrom"
  | "defaultTo"
  | "compare"
  | "trend"
  | "color"
  | "";
};

export const isTimeType = (t: Column["type"]) => {
  return (
    t === "year" ||
    t === "month" ||
    t === "hour" ||
    t === "date" ||
    t === "timestamp"
  );
};

export const isDatableType = (t: Column["type"]) => {
  return (
    t === "year" ||
    t === "month" ||
    t === "hour" ||
    t === "date" ||
    t === "timestamp" ||
    t === "time" ||
    t === "duration"
  );
};

export type GaugeCategory = {
  from: number;
  to: number;
  label?: string;
  color?: string;
};

export type MarkLine = {
  isYAxis: boolean;
  value: (string | number);
  label?: string;
};

export type Result = {
  name: string;
  visibility?: "public" | "private" | "password-protected";
  minTimeValue: number;
  maxTimeValue: number;
  reloadAt: number;
  headerImage?: string;
  footerLink?: string;
  sections: (
    | {
      type: "header";
      title?: string;
      queries: {
        render: {
          type:
          | "dropdown"
          | "dropdownMulti"
          | "button"
          | "datepicker"
          | "daterangePicker"
          | "input";
          label?: string;
        };
        columns: Column[];
        rows: (string | number | boolean)[][];
      }[];
    }
    | {
      type: "content";
      queries: {
        render:
        | {
          type:
          | "table"
          | "value"
          | "placeholder"
          | "linechart"
          | "barchartHorizontal"
          | "barchartHorizontalStacked"
          | "barchartVertical"
          | "barchartVerticalStacked"
          label?: string;
          markLines: MarkLine[];
        }
        | {
          type: "gauge";
          label?: string;
          gaugeCategories: GaugeCategory[];
        };
        columns: Column[];
        rows: (string | number | boolean)[][];
      }[];
    }
    | {
      type: "content";
      queries: {
        render: {
          type: "boxplot";
          label?: string;
          markLines: MarkLine[];
        }
        columns: Column[];
        rows: (string | number | boolean | {
          min: number;
          max: number;
          q1: number;
          q2: number;
          q3: number;
          outliers: {
            value: number;
            info?: Record<string, string> | null;
          }[];
        })[][];
      }[];
    }
  )[];
};
