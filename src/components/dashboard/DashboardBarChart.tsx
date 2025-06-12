import { Column, isTimeType, Result } from "../../lib/dashboard";
import { formatValue, formatCellValue, getIndexAxisDomain } from "../../lib/render";
import { getNameIfSet } from "../../lib/utils";
import { EChartsBarChart } from "../tremor/EChartsBarChart";

type BarProps = {
  chartId: string;
  headers: Column[];
  data: Result['sections'][0]['queries'][0]['rows']
  minTimeValue: number;
  maxTimeValue: number;
  stacked?: boolean;
  vertical?: boolean;
  label?: string;
};

const DashboardBarChart = ({
  chartId,
  headers,
  data,
  stacked,
  vertical,
  minTimeValue,
  maxTimeValue,
  label,
}: BarProps) => {
  const valueAxisIndex = headers.findIndex((c) => c.tag === "value");
  if (valueAxisIndex === -1) {
    throw new Error("No column with tag 'value'");
  }
  const valueAxisHeader = headers[valueAxisIndex]
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
  const extraDataByIndexAxis: Record<string, Record<string, [any, Column["type"]]>> = {};
  const dataByIndexAxis = data.reduce(
    (acc, row) => {
      let key = typeof row[indexAxisIndex] === 'boolean' ? row[indexAxisIndex] ? '1' : '0' : row[indexAxisIndex];
      if (key === null) {
        key = '';
      }
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
        if (i === valueAxisIndex) {
          if (categoryIndex === -1) {
            acc[key][valueAxisName] = c;
            return;
          }
          const category = (row[categoryIndex] ?? '').toString();
          categories.add(category);
          acc[key][category] = c;
          return;
        }
        const extraData = extraDataByIndexAxis[key]
        const header = headers[i]
        if (extraData != null) {
          extraData[header.name] = [c, header.type];
        } else {
          extraDataByIndexAxis[key] = { [header.name]: [c, header.type] };
        }
      });
      return acc;
    },
    {} as Record<string, Record<string, string | number>>,
  );
  const chartdata = Object.values(dataByIndexAxis);
  const indexType = isTimeType(indexAxisHeader.type) && chartdata.length < 2 ? "timestamp" : indexAxisHeader.type
  const indexAxisDomain = isTimeType(indexType)
    ? getIndexAxisDomain(minTimeValue, maxTimeValue) as [number, number]
    : indexType === "time"
      ? [minT, maxT] as [number, number]
      : undefined;

  return (
    <EChartsBarChart
      chartId={chartId}
      className="h-full select-none"
      type={stacked ? "stacked" : "default"}
      layout={vertical ? "vertical" : "horizontal"}
      data={chartdata}
      extraDataByIndexAxis={extraDataByIndexAxis}
      index={indexAxisHeader.name}
      // TODO: This logic should be in the backend in getTimestampType, but in the backend we currently do not group data by index. We should probably do the grouping also in the backend already.
      indexType={indexType}
      valueType={valueAxisHeader.type}
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
      min={indexAxisDomain ? indexAxisDomain[0] : undefined}
      max={indexAxisDomain ? indexAxisDomain[1] : undefined}
      label={label}
    />
  );
};

export default DashboardBarChart;
