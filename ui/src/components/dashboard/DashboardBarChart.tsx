// SPDX-License-Identifier: MPL-2.0

import { Column, isTimeType, Result } from "../../lib/dashboard";
import { formatValue, formatCellValue } from "../../lib/render";
import { getNameIfSet } from "../../lib/utils";
import { BarChart } from "../charts/BarChart";

type BarProps = {
  chartId: string;
  headers: Column[];
  data: Result['sections'][0]['queries'][0]['rows']
  // TODO: These are unused. We might not even need to calculate them in the backend at all.
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
  // TODO: With ECharts there should be a nicer way to show extra columns in the tooltip without aggregating them before.
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
  const indexType = indexAxisHeader.type;

  return (
    <BarChart
      chartId={chartId}
      type={stacked ? "stacked" : "default"}
      layout={vertical ? "vertical" : "horizontal"}
      data={chartdata}
      extraDataByIndexAxis={extraDataByIndexAxis}
      index={indexAxisHeader.name}
      indexType={indexType}
      valueType={valueAxisHeader.type}
      categories={Array.from(categories)}
      valueFormatter={(n: number, shortFormat?: boolean) => {
        return formatValue(n, valueAxisHeader.type, true, shortFormat).toString();
      }}
      indexFormatter={(n: number, shortFormat?: boolean) => {
        return formatValue(n, indexType, true, shortFormat).toString();
      }}
      xAxisLabel={getNameIfSet(vertical ? valueAxisName : indexAxisHeader.name)}
      yAxisLabel={getNameIfSet(vertical ? indexAxisHeader.name : valueAxisName)}
      showLegend={categoryIndex !== -1 && Array.from(categories).filter(c => c.length > 0).length > 1}
    />
  );
};

export default DashboardBarChart;
