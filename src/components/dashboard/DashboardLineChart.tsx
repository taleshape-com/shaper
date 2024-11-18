import { Column, isTimeType, Result } from "../../lib/dashboard";
import { LineChart } from "../tremor/LineChart";
import { formatValue } from "../../lib/render";

type LineProps = {
  headers: Column[];
  data: Result['sections'][0]['queries'][0]['rows']
};

const DashboardLineChart = ({ headers, data }: LineProps) => {
  const valueAxisHeader = headers.find((c) => c.tag === "value");
  if (!valueAxisHeader) {
    throw new Error("No  header with tag 'value'");
  }
  const valueAxisName = valueAxisHeader.name;
  const categoryIndex = headers.findIndex((c) => c.tag === "category");
  const categories = new Set<string>();
  if (categoryIndex === -1) {
    categories.add(valueAxisName);
  }
  const indexAxisIndex = headers.findIndex((c) => c.tag === "index");
  const indexAxisHeader = headers[indexAxisIndex];
  const dataByIndexAxis = data.reduce(
    (acc, row) => {
      const key = formatValue(row[indexAxisIndex], indexAxisHeader.type);
      if (!acc[key]) {
        acc[key] = {};
      }
      row.forEach((cell, i) => {
        if (i === indexAxisIndex || i === categoryIndex) {
          return;
        }
        const c = formatValue(cell, headers[i].type)
        if (categoryIndex === -1) {
          acc[key][valueAxisName] = c;
          return;
        }
        const category = (row[categoryIndex] ?? '').toString();
        categories.add(category);
        acc[key][category] = c;
      });
      return acc;
    },
    {} as Record<string, Record<string, string | number>>,
  );
  const chartdata = Object.entries(dataByIndexAxis).map(([key, value]) => {
    return {
      [indexAxisHeader.name]: key,
      ...value,
    };
  });

  return (
    <LineChart
      className="h-full"
      enableLegendSlider
      startEndOnly={isTimeType(indexAxisHeader.type)}
      connectNulls
      data={chartdata}
      index={indexAxisHeader.name}
      categories={Array.from(categories)}
      valueFormatter={(number: number) => {
        return number.toLocaleString();
      }}
      xAxisLabel={isTimeType(indexAxisHeader.type) ? undefined : indexAxisHeader.name}
      yAxisLabel={valueAxisName}
      showLegend={categoryIndex !== -1}
    />
  );
};

export default DashboardLineChart;
