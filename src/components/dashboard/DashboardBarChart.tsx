import { Column, isTimeType, Result } from "../../lib/dashboard";
import { formatValue, getIndexAxisDomain } from "../../lib/render";
import { getNameIfSet } from "../../lib/utils";
import { BarChart } from "../tremor/BarChart";

type BarProps = {
  chartId: string;
  headers: Column[];
  data: Result['sections'][0]['queries'][0]['rows']
  minTimeValue: number;
  maxTimeValue: number;
  stacked?: boolean;
  vertical?: boolean;
};

const DashboardBarChart = ({
  chartId,
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
            isTimeType(indexAxisHeader.type) ?
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
  const indexAxisDomain = isTimeType(indexAxisHeader.type) ? getIndexAxisDomain(minTimeValue, maxTimeValue) : undefined
  const indexType = isTimeType(indexAxisHeader.type) && chartdata.length < 2 ? "timestamp" : indexAxisHeader.type

  return (
    <BarChart
      chartId={chartId}
      className="h-full select-none"
      enableLegendSlider
      startEndOnly={chartdata.length > (vertical ? 20 : isTimeType(indexType) ? 10 : 15)}
      type={stacked ? "stacked" : "default"}
      layout={vertical ? "vertical" : "horizontal"}
      data={chartdata}
      index={indexAxisHeader.name}
      // TODO: This logic should be in the backend in getTimestampType, but in the backend we currently do not group data by index. We should probably do the grouping also in the backend already.
      indexType={indexType}
      categories={Array.from(categories)}
      valueFormatter={(n: number) => {
        return n.toLocaleString();
      }}
      indexFormatter={(n: number) => {
        return formatValue(n, indexType).toString();
      }}
      xAxisLabel={vertical ? getNameIfSet(valueAxisName) : isTimeType(indexType) ? undefined : getNameIfSet(indexAxisHeader.name)}
      yAxisLabel={vertical ? isTimeType(indexType) ? undefined : getNameIfSet(indexAxisHeader.name) : getNameIfSet(valueAxisName)}
      showLegend={categoryIndex !== -1}
      indexAxisDomain={indexAxisDomain}
    />
  );
};

export default DashboardBarChart;
