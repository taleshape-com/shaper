// SPDX-License-Identifier: MPL-2.0

import { Column, isDatableType, MarkLine, Result } from "../../lib/types";
import { Boxplot } from "../charts/Boxplot";
import { formatValue, formatCellValue } from "../../lib/render";
import { getNameIfSet } from "../../lib/utils";

type BoxplotProps = {
  chartId: string;
  label?: string;
  headers: Column[];
  data: Result["sections"][0]["queries"][0]["rows"]
  markLines?: MarkLine[];
};

const DashboardBoxplot = ({
  chartId,
  label,
  headers,
  data,
  markLines,
}: BoxplotProps) => {
  const valueAxisIndex = headers.findIndex((c) => c.tag === "value");
  if (valueAxisIndex === -1) {
    throw new Error("No  header with tag 'value'");
  }
  const valueAxisHeader = headers[valueAxisIndex];
  const valueAxisName = valueAxisHeader.name;
  const indexAxisIndex = headers.findIndex((c) => c.tag === "index");
  const indexAxisHeader = headers[indexAxisIndex];
  const colorIndex = headers.findIndex((c) => c.tag === "color");
  // TODO: With ECharts there should be a nicer way to show extra columns in the tooltip without aggregating them before.
  const extraDataByIndexAxis: Record<string, Record<string, [any, Column["type"]]>> = {};
  const boxplotData: [number, number, number, number, number][] = [];
  const outliers: [number, number, Record<string, string> | null | undefined][] = [];
  const xData: string[] = [];
  const colorByIndex = new Map<number, string>();
  data.forEach((row, rowI) => {
    let key = isDatableType(indexAxisHeader.type)
      ? new Date(row[indexAxisIndex] as number).toUTCString()
      : row[indexAxisIndex].toString();
    if (key === null) {
      if (isDatableType(indexAxisHeader.type) || indexAxisHeader.type === "number") {
        return;
      }
      key = "";
    }
    xData.push(key);
    row.forEach((cell, i) => {
      if (typeof cell === "object") {
        if (i === valueAxisIndex) {
          boxplotData.push([cell.min, cell.q1, cell.q2, cell.q3, cell.max]);
          cell.outliers.forEach(outlier => {
            outliers.push([rowI, outlier.value, outlier.info]);
          });
        }
        return;
      }
      if (i === colorIndex) {
        const color = (cell ?? "").toString();
        colorByIndex.set(rowI, color);
        return;
      }
      const c = formatCellValue(cell);
      if (i === indexAxisIndex) {
        return;
      }
      const extraData = extraDataByIndexAxis[key];
      const header = headers[i];
      if (extraData != null) {
        extraData[header.name] = [c, header.type];
      } else {
        extraDataByIndexAxis[key] = { [header.name]: [c, header.type] };
      }
    });
  });
  const indexType = indexAxisHeader.type;

  return (
    <Boxplot
      chartId={chartId}
      label={label}
      data={boxplotData}
      outliers={outliers}
      xData={xData}
      extraDataByIndexAxis={extraDataByIndexAxis}
      indexType={indexType}
      colorByIndex={colorByIndex}
      valueFormatter={(n: number, shortFormat?: boolean | number) => {
        return formatValue(n, "number", true, shortFormat).toString();
      }}
      indexFormatter={(n: number | string, shortFormat?: boolean | number) => {
        return formatValue(n, indexType, true, shortFormat).toString();
      }}
      xAxisLabel={getNameIfSet(indexAxisHeader.name)}
      yAxisLabel={getNameIfSet(valueAxisName)}
      markLines={markLines}
    />
  );
};

export default DashboardBoxplot;
