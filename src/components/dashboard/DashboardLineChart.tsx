import { Column, Result } from "../../lib/dashboard";
import { LineChart } from "../tremor/LineChart";
import { formatValue, toYear } from "../../lib/render";

type LineProps = {
  headers: Column[];
  data: Result['queries'][0]['rows']
};

const DashboardLineChart = ({ headers, data }: LineProps) => {
  const yaxisHeader = headers.find((c) => c.tag === "yAxis");
  if (!yaxisHeader) {
    throw new Error("No yaxis header found");
  }
  const yaxis = yaxisHeader.name;
  const categoryIndex = headers.findIndex((c) => c.tag === "category");
  const categories = new Set<string>();
  if (categoryIndex === -1) {
    categories.add(yaxis);
  }
  const xaxisIndex = headers.findIndex((c) => c.tag === "xAxis");
  const xaxis = headers[xaxisIndex].name;
  const dataByXaxis = data.reduce(
    (acc, row) => {
      let key = formatValue(row[xaxisIndex]);
      if (headers[xaxisIndex].type === "year") {
        key = toYear(key);
      }
      if (!acc[key]) {
        acc[key] = {};
      }
      row.forEach((cell, i) => {
        let c = formatValue(cell)
        if (i === xaxisIndex || i === categoryIndex) {
          return;
        }
        if (headers[i].type === "year") {
          c = toYear(cell);
        }
        if (categoryIndex === -1) {
          acc[key][yaxis] = c;
          return;
        }
        const category = row[categoryIndex].toString();
        categories.add(category);
        acc[key][category] = c;
      });
      return acc;
    },
    {} as Record<string, Record<string, string | number>>,
  );
  const chartdata = Object.entries(dataByXaxis).map(([key, value]) => {
    return {
      [xaxis]: key,
      ...value,
    };
  });
  return (
    <LineChart
      className="h-full w-full"
      data={chartdata}
      index={xaxis}
      categories={Array.from(categories)}
      valueFormatter={(number: number) => {
        return number.toLocaleString();
      }}
      xAxisLabel={xaxis}
      yAxisLabel={yaxis}
      showLegend={categoryIndex !== -1}
    />
  );
};

export default DashboardLineChart;
