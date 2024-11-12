export type Column = {
  name: string;
  type: "year" | "month" | "hour" | "date" | "timestamp" | "number" | "string" | "boolean";
  nullable: boolean;
  tag: "xAxis" | "yAxis" | "category" | "value" | "label" | "hint" | "download" | "";
};

export type Result = {
  title: string;
  sections: ({
    type: 'header';
    title?: string;
    queries: {
      render: {
        type: "dropdown" | "dropdownMulti" | "button";
        label?: string;
      }
      columns: Column[];
      rows: (string | number | boolean)[][];
    }[];
  } | {
    type: 'content';
    queries: {
      render: {
        type: "table" | "linechart" | "barchart";
        label?: string;
      };
      columns: Column[];
      rows: (string | number | boolean)[][];
    }[];
  })[]
};
