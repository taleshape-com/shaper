import { Column, Result } from "../../lib/dashboard";
import { LineChart } from "../tremor/LineChart";
import { formatValue, toYear } from "../../lib/render";
import { Card } from "../tremor/Card";

type LineProps = {
  label?: string;
  headers: Column[];
  data?: Result['queries'][0]['rows']
};

const DashboardLineChart = ({ label, headers, data }: LineProps) => {
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
  const dataByXaxis = (data ?? []).reduce(
    (acc, row) => {
      let key = formatValue(row[xaxisIndex]);
      if (headers[xaxisIndex].type === "year") {
        key = toYear(key);
      }
      if (!acc[key]) {
        acc[key] = {};
      }
      row.forEach((cell, i) => {
        const c = formatValue(cell)
        if (i === xaxisIndex || i === categoryIndex) {
          return;
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
    <div className="h-[calc(50vh-2rem)] p-2 mb-10">
      <h2 className="text-sm mb-2 text-center">
        {label}
      </h2>
      <Card className="h-full py-1 px-3">
        {!data ?
          (
            <div className="h-full py-1 px-3 flex items-center justify-center text-slate-600">
              no data
            </div>
          ) :
          <LineChart
            className="h-full"
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
        }
      </Card>
    </div>
  );
};

export default DashboardLineChart;
