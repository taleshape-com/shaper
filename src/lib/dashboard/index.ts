export type Column = {
  name: string;
  type: "year" | "number" | "string" | "boolean";
  nullable: boolean;
  tag: "xAxis" | "yAxis" | "category" | "value" | "label" | "hint" | "";
};

export type Result = {
  title: string;
  queries: {
    render:
    | { type: "table"; label?: string }
    | {
      type: "linechart";
      label?: string;
    }
    | {
      type: "barchart";
      label?: string;
    }
    | {
      type: "dropdown";
      label?: string;
    }
    | {
      type: "dropdownMulti";
      label?: string;
    };
    columns: Column[];
    rows: (string | number | boolean)[][];
  }[];
};
