export type Column = {
  name: string;
  type: "year" | "month" | "hour" | "date" | "timestamp" | "number" | "string" | "boolean";
  nullable: boolean;
  tag: "index" | "category" | "value" | "label" | "hint" | "download" | "default" | "defaultFrom" | "defaultTo" | "";
};

export const isTimeType = (t: Column['type']) => {
  return t === "year" || t === "month" || t === "hour" || t === "date" || t === "timestamp";
}

export type Result = {
  title: string;
  sections: ({
    type: 'header';
    title?: string;
    queries: {
      render: {
        type:
        | "dropdown"
        | "dropdownMulti"
        | "button"
        | "datepicker"
        | "daterangePicker";
        label?: string;
      }
      columns: Column[];
      rows: (string | number | boolean)[][];
    }[];
  } | {
    type: 'content';
    queries: {
      render: {
        type:
        | "table"
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
  })[]
};
