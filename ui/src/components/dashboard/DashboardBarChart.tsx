// SPDX-License-Identifier: MPL-2.0

import { Column, isTimeType, Result } from "../../lib/types";
import { formatValue, formatCellValue } from "../../lib/render";
import { getNameIfSet } from "../../lib/utils";
import { BarChart } from "../charts/BarChart";

type BarProps = {
  chartId: string;
  label?: string;
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
  label,
  headers,
  data,
  stacked,
  vertical,
}: BarProps) => {
  const valueAxisIndex = headers.findIndex((c) => c.tag === "value");
  if (valueAxisIndex === -1) {
    throw new Error("No column with tag 'value'");
  }
  const colorIndex = headers.findIndex((c) => c.tag === "color");
  const valueAxisHeader = headers[valueAxisIndex]
  const valueAxisName = valueAxisHeader.name;
  const categoryIndex = headers.findIndex((c) => c.tag === "category");
  const categories = new Set<string>();
  const colorsByCategory = {} as Record<string, string>;
  if (categoryIndex === -1) {
    categories.add(valueAxisName);
  }
  const indexAxisIndex = headers.findIndex((c) => c.tag === "index");
  const indexAxisHeader = headers[indexAxisIndex];
  // TODO: With ECharts there should be a nicer way to show extra columns in the tooltip without aggregating them before.
  const extraDataByIndexAxis: Record<string, Record<string, [any, Column["type"]]>> = {};
  const dataByIndexAxis = new Map<string | number, Record<string, string | number>>();
  data.forEach((row) => {
    let key = typeof row[indexAxisIndex] === 'boolean' ? row[indexAxisIndex] ? '1' : '0' : row[indexAxisIndex];
    if (key === null) {
      if (isTimeType(indexAxisHeader.type)) {
        return
      }
      key = '';
    }
    if (!dataByIndexAxis.get(key)) {
      dataByIndexAxis.set(key, {
        [indexAxisHeader.name]:
          isTimeType(indexAxisHeader.type) ?
            (new Date(key)).getTime() : key,
      });
    }
    const v = dataByIndexAxis.get(key);
    if (v == null) {
      return;
    }
    row.forEach((cell, i) => {
      if (i === indexAxisIndex) {
        return;
      }
      if (i === categoryIndex) {
        return;
      }
      if (i === colorIndex) {
        const color = (cell ?? '').toString();
        if (color.length > 0) {
          if (categoryIndex === -1) {
            colorsByCategory[valueAxisName] = color;
          } else {
            const category = (row[categoryIndex] ?? '').toString();
            colorsByCategory[category] = color;
          }
        }
        return;
      }
      const c = formatCellValue(cell)
      if (i === valueAxisIndex) {
        if (categoryIndex === -1) {
          v[valueAxisName] = c;
          return;
        }
        const category = (row[categoryIndex] ?? '').toString();
        categories.add(category);
        v[category] = c;
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
    return dataByIndexAxis;
  });
  const indexType = indexAxisHeader.type;

  return (
    <BarChart
      chartId={chartId}
      label={label}
      type={stacked ? "stacked" : "default"}
      layout={vertical ? "vertical" : "horizontal"}
      data={Array.from(dataByIndexAxis.values())}
      extraDataByIndexAxis={extraDataByIndexAxis}
      index={indexAxisHeader.name}
      indexType={indexType}
      valueType={valueAxisHeader.type}
      categories={Array.from(categories)}
      colorsByCategory={colorsByCategory}
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
