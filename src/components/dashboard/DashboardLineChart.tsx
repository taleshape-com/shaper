import { Column } from "../../lib/dashboard";
import { LineChart } from "../tremor/LineChart";

type LineProps = {
  headers: Column[];
  data: (string | number)[][];
  xaxis: string;
};

const toYear = (value: string | number) => {
  return new Date(value).getFullYear();
};

const DashboardLineChart = ({ headers, data, xaxis }: LineProps) => {
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

export default DashboardLineChart;
