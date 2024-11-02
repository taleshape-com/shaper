import { useState, useEffect } from "react";
import "./App.css";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeaderCell,
  TableRoot,
  TableRow,
} from "./components/Table";
import { LineChart } from "./components/LineChart";

type Column = {
  name: string;
  type: "year" | "number" | "string";
  nullable: boolean;
};

type TableProps = {
  headers: Column[];
  data: (string | number)[][];
};

type LineProps = {
  headers: Column[];
  data: (string | number)[][];
  xaxis: string;
};

type Result = {
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

const toYear = (value: string | number) => {
  return new Date(value).getFullYear();
};

const formatValue = (value: string | number): string => {
  if (typeof value === "number") {
    return value.toLocaleString();
  }
  return value;
};

function MyTable({ headers, data }: TableProps) {
  return (
    <TableRoot className="max-h-screen overflow-auto w-full">
      <Table>
        <TableHead>
          <TableRow>
            {headers.map((header) => (
              <TableHeaderCell key={header.name}>{header.name}</TableHeaderCell>
            ))}
          </TableRow>
        </TableHead>
        <TableBody>
          {data.map((items, index) => (
            <TableRow key={index}>
              {items.map((item, index) => {
                const classes =
                  typeof item === "number" ? "text-right" : "text-left";
                return (
                  <TableCell key={index} className={classes}>
                    {formatValue(item)}
                  </TableCell>
                );
              })}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableRoot>
  );
}

export const MyLine = ({ headers, data, xaxis }: LineProps) => {
  const chartdata = data.map((row) => {
    const d = {} as Record<string, string | number>;
    headers.forEach((c, i) => {
      if (c.type === "year") {
        d[c.name] = toYear(row[i]);
        return;
      }
      d[c.name] = row[i];
    });
    return d;
  });
  const categories = headers.map((h) => h.name).filter((h) => h !== xaxis);
  return (
    <LineChart
      className="h-full w-full"
      data={chartdata}
      index={xaxis}
      categories={categories}
      valueFormatter={(number: number) => {
        return number.toLocaleString();
      }}
      xAxisLabel={xaxis}
      yAxisLabel={categories.length === 1 ? categories[0] : undefined}
      showLegend={categories.length > 1}
    />
  );
};

function App() {
  const [data, setData] = useState<Result>({
    title: "Loading...",
    queries: [],
  });
  useEffect(() => {
    fetch("http://localhost:1323/api/sample")
      .then((response) => response.json())
      .then((fetchedData: Result) => {
        setData(fetchedData);
      })
      .catch((error) => console.error("Error fetching data:", error));
  }, []);
  let nextTitle: string | undefined = undefined;
  return (
    <div className="w-screen h-screen px-4 py-8 sm:px-6 lg:px-8 overflow-auto">
      <h1 className="mb-8 text-xl text-center">{data.title}</h1>
      <div className="flex flex-col lg:flex-row lg:flex-wrap gap-4">
        {data.queries.length === 0 ? (
          <div>No data to show...</div>
        ) : (
          data.queries.map(({ render, columns, rows }, index) => {
            if (render.type === "title") {
              nextTitle = rows[0][0] as string;
              return;
            }
            let title: string | undefined = undefined;
            if (nextTitle) {
              title = nextTitle;
              nextTitle = undefined;
            }
            return (
              <div
                key={index}
                className="lg:w-[calc(50vw-5rem)] h-[calc(50vh-4rem)] lg:h-[calc(100vh-12rem)]"
              >
                <h2 className="text-lg mb-10 text-center">{title}</h2>
                {render.type === "line" ? (
                  <MyLine headers={columns} data={rows} xaxis={render.xAxis} />
                ) : (
                  <MyTable headers={columns} data={rows} />
                )}
              </div>
            );
          })
        )}
      </div>
    </div>
  );
}

export default App;
