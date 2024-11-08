export type Column = {
  name: string;
  type: "year" | "number" | "string";
  nullable: boolean;
};

export type Result = {
  title: string;
  queries: {
    render:
      | { type: "table"; label?: string }
      | {
          type: "linechart";
          label?: string;
          xAxis: string;
          yAxis: string;
          categoryIndex?: number;
        }
      | {
          type: "barchart";
          label?: string;
          xAxis: string;
          yAxis: string;
          categoryIndex?: number;
        };
    columns: {
      name: string;
      type: "year" | "number" | "string";
      nullable: boolean;
    }[];
    rows: (string | number)[][];
  }[];
};
