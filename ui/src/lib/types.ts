// SPDX-License-Identifier: MPL-2.0

export type QueryExecutionStatus = "pending" | "running" | "success" | "failed" | "cancelled" | "timed_out";
export type QueryExecutionType = "dashboard" | "task" | "sql_api" | "download";

export interface QueryExecution {
  id: string;
  type: QueryExecutionType;
  dashboardId?: string;
  taskId?: string;
  userId?: string;
  apiKeyId?: string;
  queryIndex?: number;
  query?: string;
  status: QueryExecutionStatus;
  startedAt: string;
  endedAt?: string;
  durationMs?: number;
  rowCount?: number;
  error?: string;
  isSlowQuery: boolean;
}

export const SLOW_QUERY_THRESHOLD_MS = 1000;

export interface QuerySummary {
  durationMs: number;
  rowCount: number;
  status: QueryExecutionStatus;
  isSlowQuery: boolean;
}

export interface IApp {
  id: string;
  name: string;
  path?: string;
  folderId?: string;
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

export type IDashboard = Omit<IApp, "type"> & {
  content: string;
};

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
  | "small"
  | "medium"
  | "large"
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
  querySummary?: QuerySummary;
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
          | "piechart"
          | "donutchart"
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
