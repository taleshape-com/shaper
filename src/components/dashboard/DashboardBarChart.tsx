import { Column, isTimeType, Result } from "../../lib/dashboard";
import { formatValue, formatCellValue, getIndexAxisDomain } from "../../lib/render";
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
  // TODO: It might more efficient to calculate these min/max values in the backend
  let minT = Number.MAX_VALUE;
  let maxT = 0;
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
        if (i === indexAxisIndex) {
          if (indexAxisHeader.type === 'time' && typeof cell === 'number') {
            if (cell < minT) {
              minT = cell;
            }
            if (cell > maxT) {
              maxT = cell;
            }
          }
          return;
        }
        if (i === categoryIndex) {
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
  const indexAxisDomain = isTimeType(indexType) ? getIndexAxisDomain(minTimeValue, maxTimeValue) : indexType === "time" ? [minT, maxT] : undefined

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
        return formatValue(n, valueAxisHeader.type, true).toString();
      }}
      indexFormatter={(n: number) => {
        return formatValue(n, indexType, true).toString();
      }}
      xAxisLabel={getNameIfSet(vertical ? valueAxisName : indexAxisHeader.name)}
      yAxisLabel={getNameIfSet(vertical ? indexAxisHeader.name : valueAxisName)}
      showLegend={categoryIndex !== -1}
      indexAxisDomain={indexAxisDomain}
      maxValue={valueAxisHeader.type === 'percent' ? 1 : undefined}
    />
  );
};

export default DashboardBarChart;
