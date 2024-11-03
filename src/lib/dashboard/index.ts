export type Column = {
  name: string;
  type: "year" | "number" | "string";
  nullable: boolean;
};

export type Result = {
  title: string;
  queries: {
    render:
      | { type: "title" }
      | { type: "table" }
      | { type: "line"; xAxis: string };
    columns: {
      name: string;
      type: "year" | "number" | "string";
      nullable: boolean;
    }[];
    rows: (string | number)[][];
  }[];
};
