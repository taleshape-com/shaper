import { Column } from "../../lib/dashboard";
import { BarChart } from "../tremor/BarChart";
import { LineChart } from "../tremor/LineChart";

type LineProps = {
  headers: Column[];
  data: (string | number)[][];
  xaxis: string;
  yaxis: string;
  categoryIndex?: number;
};

const toYear = (value: string | number) => {
  return new Date(value).getFullYear();
};

const DashboardBarChart = ({
  headers,
  data,
  xaxis,
  yaxis,
  categoryIndex,
}: LineProps) => {
  const categories = new Set<string>();
  if (categoryIndex == null) {
    categories.add(yaxis);
  }
  const xaxisIndex = headers.findIndex((c) => c.name === xaxis);
  const dataByXaxis = data.reduce(
    (acc, row) => {
      const key = row[xaxisIndex];
      if (!acc[key]) {
        acc[key] = {};
      }
      row.forEach((cell, i) => {
        if (i === xaxisIndex && i === categoryIndex) {
          return;
        }
        // if (c.type === "year") {
        //   d[c.name] = toYear(row[i]);
        //   return;
        // }
        if (categoryIndex == null) {
          acc[key][yaxis] = cell;
          return;
        }
        const category = row[categoryIndex].toString();
        categories.add(category);
        acc[key][category] = cell;
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
    <BarChart
      className="h-full w-full"
      type="stacked"
      data={chartdata}
      index={xaxis}
      categories={Array.from(categories)}
      valueFormatter={(number: number) => {
        return number.toLocaleString();
      }}
      xAxisLabel={xaxis}
      yAxisLabel={yaxis}
      showLegend={categoryIndex != null}
    />
  );
};

export default DashboardBarChart;
