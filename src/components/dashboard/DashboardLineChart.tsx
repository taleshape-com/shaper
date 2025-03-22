import { Column, isTimeType, Result } from "../../lib/dashboard";
import { LineChart } from "../tremor/LineChart";
import { formatValue, formatCellValue, getIndexAxisDomain } from "../../lib/render";
import { getNameIfSet } from "../../lib/utils";

type LineProps = {
  chartId: string;
  headers: Column[];
  data: Result['sections'][0]['queries'][0]['rows']
  minTimeValue: number;
  maxTimeValue: number;
};

const DashboardLineChart = ({
  chartId,
  headers,
  data,
  minTimeValue,
  maxTimeValue,
}: LineProps) => {
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
        const c = formatCellValue(cell)
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
  const indexType = isTimeType(indexAxisHeader.type) && chartdata.length < 2 ? "timestamp" : indexAxisHeader.type
  const xAxisDomain = isTimeType(indexType) ? getIndexAxisDomain(minTimeValue, maxTimeValue) : undefined

  return (
    <LineChart
      chartId={chartId}
      className="h-full select-none"
      enableLegendSlider
      startEndOnly={isTimeType(indexType)}
      connectNulls
      data={chartdata}
      index={indexAxisHeader.name}
      // TODO: This logic should be in the backend in getTimestampType, but in the backend we currently do not group data by index. We should probably do the grouping also in the backend already.
      indexType={indexType}
      categories={Array.from(categories)}
      valueFormatter={(n: number) => {
        return formatValue(n, valueAxisHeader.type, true).toString();
      }}
      indexFormatter={(n: number) => {
        return formatValue(n, indexType, true).toString();
      }}
      xAxisLabel={getNameIfSet(indexAxisHeader.name)}
      yAxisLabel={getNameIfSet(valueAxisName)}
      showLegend={categoryIndex !== -1}
      xAxisDomain={xAxisDomain}
    />
  );
};

export default DashboardLineChart;
