// SPDX-License-Identifier: MPL-2.0

import { Column } from "../../lib/types";
import { formatValue, formatCellValue } from "../../lib/render";
import { PieChart } from "../charts/PieChart";
import { useCallback, useMemo } from "react";
import { getNameIfSet } from "../../lib/utils";

type PieProps = {
  chartId: string;
  label?: string;
  headers: Column[];
  data: (string | number | boolean)[][];
  isDonut?: boolean;
};

const DashboardPieChart = ({
  chartId,
  label,
  headers,
  data,
  isDonut = false,
}: PieProps) => {
  const valueIndex = headers.findIndex((c) => c.tag === "value");
  if (valueIndex === -1) {
    throw new Error("No column with tag 'value'");
  }
  const valueHeader = headers[valueIndex];

  const categoryIndex = headers.findIndex((c) => c.tag === "category");
  const colorIndex = headers.findIndex((c) => c.tag === "color");

  // Calculate extra data by name (category)
  const extraDataByName = useMemo(() => {
    const extraData: Record<string, Record<string, [any, Column["type"]]>> = {};

    data.forEach((row) => {
      const name =
        categoryIndex !== -1
          ? (row[categoryIndex] ?? "").toString()
          : (getNameIfSet(valueHeader.name) ?? "");

      row.forEach((cell, i) => {
        // Skip value, category, and color columns
        if (i === valueIndex || i === categoryIndex || i === colorIndex) {
          return;
        }

        const header = headers[i];
        const extraDataForName = extraData[name];
        const formattedValue = formatCellValue(cell);

        if (extraDataForName != null) {
          extraDataForName[header.name] = [formattedValue, header.type];
        } else {
          extraData[name] = { [header.name]: [formattedValue, header.type] };
        }
      });
    });

    return extraData;
  }, [data, headers, valueIndex, categoryIndex, colorIndex, valueHeader]);

  // Transform data into pie chart format
  const pieData = useMemo(() => {
    return data.map(row => {
      const value = formatCellValue(row[valueIndex]) as number;
      const name =
        categoryIndex !== -1
          ? (row[categoryIndex] ?? "").toString()
          : (getNameIfSet(valueHeader.name) ?? "");
      const color =
        colorIndex !== -1 ? (row[colorIndex] ?? "").toString() : undefined;

      return {
        name,
        value,
        color: color && color.length > 0 ? color : undefined,
      };
    });
  }, [data, valueHeader, categoryIndex, colorIndex, valueIndex]);

  const valueFormatter = useCallback(
    (n: number) => formatValue(n, valueHeader.type, true).toString(),
    [valueHeader.type],
  );

  return (
    <PieChart
      chartId={chartId}
      label={label}
      data={pieData}
      extraDataByName={extraDataByName}
      valueType={valueHeader.type}
      valueColumnName={getNameIfSet(valueHeader.name)}
      valueFormatter={valueFormatter}
      isDonut={isDonut}
    />
  );
};

export default DashboardPieChart;
