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
import { LineChart } from "./components/LineChart"


const isYear = (value: any) => {
  return typeof value === "string" && value.match(/^\d{4}-01-01T00:00:00Z$/);
}

const toYear = (value: string) => {
  return new Date(value).getFullYear();
}

const formatValue = (value: any) => {
  if (typeof value === "number") {
    return value.toLocaleString();
  }
  if (isYear(value)) {
    return toYear(value);
  }
  return value;
};

const isGraphData = (data: any[][]) => {
  return data.every((row) => {
    if (row.length < 2) {
      return false
    }
    const [first, ...rest] = row;
    return  new Date(first).toString() !== "Invalid Date" && rest.every((v) => typeof v === "number");
  })
}
function MyTable({ headers, data }: { headers: string[]; data: any[][] }) {
  return <TableRoot className="max-h-screen overflow-auto w-full">
      <Table>
      <TableHead>
        <TableRow>
          {headers.map((header) => (
            <TableHeaderCell key={header}>{header}</TableHeaderCell>
          ))}
        </TableRow>
      </TableHead>
      <TableBody>
        {data.map((items, index) => (
          <TableRow key={index}>
            {items.map((item, index) => {
              const classes = typeof item === "number" ? "text-right" : "text-left";
              return <TableCell key={index} className={classes}>
                {formatValue(item)}
              </TableCell>
            })}
          </TableRow>
        ))}
      </TableBody>
    </Table>
  </TableRoot>
}

export const MyGraph = ({ headers, data }: { headers: string[]; data: any[][] }) => {
  const [index, ...categories] = headers
  const chartdata = data.map((row) => {
    const d = {} as Record<string, any>;
    headers.forEach((c, i) => {
      if (i === 0 && isYear(row[0])) {
        d[c] = toYear(row[0]);
        return;
      }
      d[c] = row[i];
    });
    return d;
  });
  return <LineChart
    className="h-full w-full"
    data={chartdata}
    index={index}
    categories={categories}
    valueFormatter={(number: number) => {
        return number.toLocaleString();
    }}
    xAxisLabel={index}
    yAxisLabel={categories.length === 1 ? categories[0] :undefined}
    showLegend={categories.length > 1}
  />
}

type Result = {
  title: string;
  queries: {
    columns: string[];
    rows: any[][];
  }[]
};

function isText(rows: any[][]) {
	return rows.length === 1 && rows[0].length === 1 && typeof rows[0][0] === "string";
}

function App() {
  const [data, setData] = useState<Result>({ title: "Loading...", queries: [] });
  useEffect(() => {
    fetch("http://localhost:1323/api/sample")
      .then((response) => response.json())
      .then((fetchedData: Result) => {
        setData(fetchedData);
      })
      .catch((error) => console.error("Error fetching data:", error));
  }, []);
  return (
    <div className="w-screen h-screen px-4 py-8 sm:px-6 lg:px-8 overflow-auto">
      <h1 className="mb-8 text-xl text-center">{data.title}</h1>
      <div className="flex flex-col lg:flex-row lg:flex-wrap gap-4">
        {data.queries.length === 0 ? (
          <div>No data to show...</div>
        ) : (
					data.queries.map(({ columns, rows }, index) => {
						if (isText(rows)) {
							return <span>{rows[0][0]}</span>
						}
						return <div key={index} className="lg:w-[calc(50vw-5rem)] h-[calc(50vh-4rem)] lg:h-[calc(100vh-12rem)]">
							{isGraphData(rows) ? (
								<MyGraph headers={columns} data={rows} />
								) : (
									<MyTable headers={columns} data={rows} />
								)}
							</div>
							})
        )}
      </div>
    </div>
  );
}

export default App;
