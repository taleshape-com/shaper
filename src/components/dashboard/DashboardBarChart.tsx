import { Column, isTimeType, Result } from "../../lib/dashboard";
import { formatValue, getXAxisDomain } from "../../lib/render";
import { BarChart } from "../tremor/BarChart";

type BarProps = {
  headers: Column[];
  data: Result['sections'][0]['queries'][0]['rows']
  minTimeValue: number;
  maxTimeValue: number;
  stacked?: boolean;
  vertical?: boolean;
};

const DashboardBarChart = ({
  headers,
  data,
  stacked,
  vertical,
  minTimeValue,
  maxTimeValue,
}: BarProps) => {
  const valueAxisHeader = headers.find((c) => c.tag === "value");
  if (!valueAxisHeader) {
    throw new Error("No column with tag 'value'");
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
      const key = typeof row[indexAxisIndex] === 'boolean' ? row[indexAxisIndex] ? '1' : '0' : row[indexAxisIndex];
      if (!acc[key]) {
        acc[key] = {
          [indexAxisHeader.name]:
            indexAxisHeader.type === 'year' ||
              indexAxisHeader.type === 'month' ||
              indexAxisHeader.type === 'date' ||
              indexAxisHeader.type === 'timestamp' ||
              indexAxisHeader.type === 'hour' ?
              (new Date(key)).getTime() : key,
        };
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
  const chartdata = Object.values(dataByIndexAxis);
  const xAxisDomain = isTimeType(indexAxisHeader.type) ? getXAxisDomain(minTimeValue, maxTimeValue) : undefined

  return (
    <BarChart
      className="h-full"
      enableLegendSlider
      startEndOnly={chartdata.length > (vertical ? 20 : isTimeType(indexAxisHeader.type) ? 10 : 15)}
      type={stacked ? "stacked" : "default"}
      layout={vertical ? "vertical" : "horizontal"}
      data={chartdata}
      index={indexAxisHeader.name}
      categories={Array.from(categories)}
      valueFormatter={(n: number) => {
        return n.toLocaleString();
      }}
      indexFormatter={(n: number) => {
        return formatValue(n, indexAxisHeader.type).toString();
      }}
      xAxisLabel={vertical ? valueAxisName : isTimeType(indexAxisHeader.type) ? undefined : indexAxisHeader.name}
      yAxisLabel={vertical ? isTimeType(indexAxisHeader.type) ? undefined : indexAxisHeader.name : valueAxisName}
      showLegend={categoryIndex !== -1}
      xAxisDomain={xAxisDomain}
    />
  );
};

export default DashboardBarChart;
