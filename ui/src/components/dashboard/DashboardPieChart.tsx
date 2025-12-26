// SPDX-License-Identifier: MPL-2.0

import { Column } from "../../lib/types";
import { formatValue, formatCellValue } from "../../lib/render";
import { PieChart } from "../charts/PieChart";

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

  // Transform data into pie chart format
  const pieData = data.map((row, rowIndex) => {
    const value = formatCellValue(row[valueIndex]) as number;
    const name =
      categoryIndex !== -1
        ? (row[categoryIndex] ?? "").toString()
        : `${valueHeader.name} ${rowIndex + 1}`;
    const color =
      colorIndex !== -1 ? (row[colorIndex] ?? "").toString() : undefined;

    return {
      name,
      value,
      color: color && color.length > 0 ? color : undefined,
    };
  });

  return (
    <PieChart
      chartId={chartId}
      label={label}
      data={pieData}
      valueType={valueHeader.type}
      valueFormatter={(n: number) =>
        formatValue(n, valueHeader.type, true).toString()
      }
      showLegend={pieData.length > 1 && pieData.length <= 10}
      isDonut={isDonut}
    />
  );
};

export default DashboardPieChart;
