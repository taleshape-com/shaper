export interface IDashboard {
  id: string;
  name: string;
  path: string;
  content: string;
  createdAt: string;
  updatedAt: string;
  createdBy?: string;
  updatedBy?: string;
  visibility?: 'public' | 'private';
}

export type Column = {
  name: string;
  type: "year" | "month" | "hour" | "date" | "timestamp" | "duration" | "time" | "number" | "string" | "boolean" | "object" | "array" | "percent";
  nullable: boolean;
  tag: "index" | "category" | "value" | "label" | "hint" | "download" | "default" | "defaultFrom" | "defaultTo" | "compare" | "trend" | "";
};

export const isTimeType = (t: Column['type']) => {
  return t === "year" || t === "month" || t === "hour" || t === "date" || t === "timestamp";
}

export type GaugeCategory = {
  from: number;
  to: number;
  label?: string;
  color?: string;
};

export type Result = {
  name: string;
  minTimeValue: number;
  maxTimeValue: number;
  reloadAt: number;
  sections: ({
    type: 'header';
    title?: string;
    queries: ({
      render:
      | ({
        type:
        | "dropdown"
        | "dropdownMulti"
        | "button"
        | "datepicker"
        | "daterangePicker";
        label?: string;
      })
      | ({
        type: "gauge";
        label?: string;
        gaugeCategories: GaugeCategory[];
      });
      columns: Column[];
      rows: (string | number | boolean)[][];
    })[];
  } | {
    type: 'content';
    queries: {
      render: {
        type:
        | "table"
        | "value"
        | "placeholder"
        | "linechart"
        | "barchartHorizontal"
        | "barchartHorizontalStacked"
        | "barchartVertical"
        | "barchartVerticalStacked";
        label?: string;
      };
      columns: Column[];
      rows: (string | number | boolean)[][];
    }[];
  })[];
};
