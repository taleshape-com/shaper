// SPDX-License-Identifier: MPL-2.0

import { Column } from "../../lib/types";
import { formatValue, formatCellValue } from "../../lib/render";
import { PieChart } from "../charts/PieChart";
import { useCallback, useMemo } from "react";
import { getNameIfSet } from "../../lib/utils";
import { translate } from "../../lib/translate";

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

    // Calculate all original data first
    const allData = data.map(row => {
      const value = formatCellValue(row[valueIndex]) as number;
      const name =
        categoryIndex !== -1
          ? (row[categoryIndex] ?? "").toString()
          : (getNameIfSet(valueHeader.name) ?? "");

      return {
        name,
        value,
      };
    });

    // Calculate total value to determine percentages
    const totalValue = allData.reduce((sum, item) => sum + item.value, 0);

    // Separate items that are >= 5% from those that are < 5%
    const significantItems = allData.filter(item => totalValue > 0 && (item.value / totalValue) * 100 >= 5);
    const smallItems = allData.filter(item => totalValue > 0 && (item.value / totalValue) * 100 < 5);

    // Process original data to create extra data for significant items
    data.forEach((row) => {
      const name =
        categoryIndex !== -1
          ? (row[categoryIndex] ?? "").toString()
          : (getNameIfSet(valueHeader.name) ?? "");

      // Only add extra data for significant items (not for "Other")
      const isSignificant = significantItems.some(item => item.name === name);
      if (!isSignificant) {
        return; // Skip extra data for small items that will be combined into "Other"
      }

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

    // If there are multiple small items, create special extra data for "Other" category
    if (smallItems.length > 1) {
      // Create a summary of values for each category in "Other"
      const otherExtraData: Record<string, [any, Column["type"]]> = {};
      smallItems.forEach(item => {
        otherExtraData[item.name] = [item.value, valueHeader.type];
      });

      extraData[translate("Other")] = otherExtraData;
    } else if (smallItems.length === 1) {
      // For single small items, add normal extra data like other significant items
      data.forEach((row) => {
        const name =
          categoryIndex !== -1
            ? (row[categoryIndex] ?? "").toString()
            : (getNameIfSet(valueHeader.name) ?? "");

        // Add extra data for the single small item
        const isSmall = smallItems.some(item => item.name === name);
        if (isSmall) {
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
        }
      });
    }

    return extraData;
  }, [data, headers, valueIndex, categoryIndex, colorIndex, valueHeader]);

  // Transform data into pie chart format
  const pieData = useMemo(() => {
    // First, calculate all values to determine percentages
    const allData = data.map(row => {
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

    // Calculate total value to determine percentages
    const totalValue = allData.reduce((sum, item) => sum + item.value, 0);

    if (totalValue === 0) {
      return allData;
    }

    // Separate items that are >= 5% from those that are < 5%
    const significantItems = allData.filter(item => (item.value / totalValue) * 100 >= 5);
    const smallItems = allData.filter(item => (item.value / totalValue) * 100 < 5);

    // If there's only one small item, keep it as is instead of combining into "Other"
    if (smallItems.length === 0) {
      return allData;
    } else if (smallItems.length === 1) {
      // Return all items as they are if there's only one small item
      return allData;
    }

    // Combine small items into "Other" category if there are multiple small items
    const otherValue = smallItems.reduce((sum, item) => sum + item.value, 0);
    const otherItem = {
      name: translate("Other"),
      value: otherValue,
      color: undefined, // Use default color for "Other"
    };

    // Return significant items plus the "Other" item
    return [...significantItems, otherItem];
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
