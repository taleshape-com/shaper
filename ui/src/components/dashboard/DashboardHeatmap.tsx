// SPDX-License-Identifier: MPL-2.0

import { useCallback, useMemo } from "react";
import { Column } from "../../lib/types";
import { HeatmapChart } from "../charts/HeatmapChart";
import { formatValue, formatCellValue } from "../../lib/render";
import { getNameIfSet } from "../../lib/utils";

type HeatmapProps = {
  chartId: string;
  label?: string;
  headers: Column[];
  data: (string | number | boolean)[][];
};

const pad2 = (n: number) => n.toString().padStart(2, "0");

const toIsoDate = (value: string | number | boolean | null | undefined): string | null => {
  if (value === null || value === undefined) {
    return null;
  }
  // Backend serializes time-typed columns as Unix ms numbers (or pre-formatted strings).
  const d = new Date(value as string | number);
  if (Number.isNaN(d.getTime())) {
    return null;
  }
  // Use UTC components — backend interprets dates in UTC for consistency across timezones.
  return `${d.getUTCFullYear()}-${pad2(d.getUTCMonth() + 1)}-${pad2(d.getUTCDate())}`;
};

const DashboardHeatmap = ({
  chartId,
  label,
  headers,
  data,
}: HeatmapProps) => {
  const valueIndex = headers.findIndex((c) => c.tag === "value");
  if (valueIndex === -1) {
    throw new Error("No column with tag 'value'");
  }
  const indexAxisIndex = headers.findIndex((c) => c.tag === "index");
  if (indexAxisIndex === -1) {
    throw new Error("No column with tag 'index'");
  }
  const valueHeader = headers[valueIndex];

  const heatmapData = useMemo(() => {
    const out: [string, number][] = [];
    data.forEach((row) => {
      const date = toIsoDate(row[indexAxisIndex] as string | number);
      if (!date) {
        return;
      }
      const raw = formatCellValue(row[valueIndex]);
      const value = typeof raw === "number" ? raw : Number(raw);
      if (!Number.isFinite(value)) {
        return;
      }
      out.push([date, value]);
    });
    return out;
  }, [data, indexAxisIndex, valueIndex]);

  const valueFormatter = useCallback(
    (n: number) => formatValue(n, valueHeader.type, true).toString(),
    [valueHeader.type],
  );

  return (
    <HeatmapChart
      chartId={chartId}
      label={label}
      data={heatmapData}
      valueColumnName={getNameIfSet(valueHeader.name)}
      valueFormatter={valueFormatter}
    />
  );
};

export default DashboardHeatmap;
