// SPDX-License-Identifier: MPL-2.0

import React, { useCallback, useRef } from "react";
import type { ECharts } from "echarts/core";
import type { PieSeriesOption } from "echarts/charts";
import {
  constructCategoryColors,
  getThemeColors,
  getChartFont,
  getDisplayFont,
} from "../../lib/chartUtils";
import { cx } from "../../lib/utils";
import { DarkModeContext } from "../../contexts/DarkModeContext";
import { Column } from "../../lib/types";
import { formatValue, echartsEncode } from "../../lib/render";
import { translate } from "../../lib/translate";
import { EChart } from "./EChart";

const chartPadding = 16;

interface PieChartProps extends React.HTMLAttributes<HTMLDivElement> {
  chartId: string;
  label?: string;
  data: { name: string; value: number; color?: string }[];
  extraDataByName: Record<string, Record<string, any>>;
  valueType: Column["type"];
  valueColumnName?: string;
  valueFormatter: (value: number) => string;
  isDonut?: boolean;
}

const PieChart = (props: PieChartProps) => {
  const {
    data,
    extraDataByName,
    valueFormatter,
    className,
    chartId,
    label,
    isDonut = false,
    ...other
  } = props;

  const chartRef = useRef<ECharts | null>(null);
  const [chartWidth, setChartWidth] = React.useState(450);
  const [chartHeight, setChartHeight] = React.useState(300);
  const { isDarkMode } = React.useContext(DarkModeContext);

  const chartOptions = React.useMemo(() => {
    const theme = getThemeColors(isDarkMode);
    const chartFont = getChartFont();
    const displayFont = getDisplayFont();

    const categories = data.map((d) => d.name);
    const colorsByCategory: Record<string, string> = {};
    data.forEach((d) => {
      if (d.color) colorsByCategory[d.name] = d.color;
    });
    const categoryColors = constructCategoryColors(
      categories,
      colorsByCategory,
      isDarkMode,
    );

    const labelTopOffset = label
      ? 40 + 15 * (Math.ceil(label.length / (0.125 * chartWidth)) - 1)
      : 0;

    // Calculate center position to account for title
    const availableHeight = chartHeight - labelTopOffset - chartPadding * 2;
    const centerY = labelTopOffset + chartPadding + availableHeight * 0.51;
    const radius = Math.min(Math.min(chartWidth, chartHeight), 800) * 0.32;

    const series: PieSeriesOption = {
      type: "pie",
      radius: isDonut ? [radius * 0.55, radius] : radius,
      center: ["50%", centerY],
      data: data.map((d) => ({
        name: d.name,
        value: d.value,
        itemStyle: {
          color: categoryColors.get(d.name),
        },
      })),
      label: {
        show: chartWidth > 350 && data.length <= 8,
        fontFamily: chartFont,
        color: theme.textColorSecondary,
        fontSize: 12,
        fontWeight: 500,
      },
      labelLine: {
        show: false,
        length: 12,
        length2: 0,
      },
      itemStyle: {
        borderRadius: 2,
        borderColor: theme.backgroundColorSecondary,
        borderWidth: isDonut ? 2 : 1.5,
      },
      animationDelay: 100,
      animationDelayUpdate: 100,
      cursor: "crosshair",
    };

    const titles: any[] = [
      {
        text: label,
        textStyle: {
          fontSize: 16,
          lineHeight: 16,
          fontFamily: displayFont,
          fontWeight: 600,
          color: theme.textColor,
          width: chartWidth - 10 - 2 * chartPadding,
          overflow: "break",
        },
        left: "center",
        top: chartPadding,
      },
    ];

    if (isDonut) {
      const totalValue = data.reduce((acc, d) => acc + d.value, 0);
      const formattedTotal = valueFormatter(totalValue);
      if (formattedTotal.length < 10) {
        titles.push({
          text: `{val|${formattedTotal}}\n{label|${translate("Total")}}`,
          left: "center",
          top: centerY * 0.96,
          textStyle: {
            rich: {
              val: {
                fontSize: 20,
                fontWeight: 600,
                fontFamily: displayFont,
                color: theme.textColor,
                lineHeight: 30,
              },
              label: {
                fontSize: 14,
                fontWeight: 400,
                fontFamily: chartFont,
                color: theme.textColorSecondary,
                lineHeight: 16,
              },
            },
          },
          textVerticalAlign: "middle",
        });
      }
    }

    return {
      title: titles,
      tooltip: {
        trigger: "item",
        confine: true,
        backgroundColor: undefined,
        borderColor: theme.borderColor,
        className: "bg-cbg dark:bg-dbg",
        textStyle: {
          fontFamily: chartFont,
          color: theme.textColor,
        },
        formatter: (params: any) => {
          const percentage = params.percent.toFixed(1);
          let tooltipContent = `<div class="text-sm">
            <div class="flex items-center space-x-2">
              <span class="inline-block size-2 rounded-sm" style="background-color: ${echartsEncode(params.color)}"></span>
              <span class="font-medium">${echartsEncode(params.name)}</span>
            </div>`;

          // Show value with its column name if available
          const formattedValue = echartsEncode(valueFormatter(params.value));
          if (props.valueColumnName) {
            const valueColumnName = echartsEncode(props.valueColumnName);
            tooltipContent += `<div class="mt-1 flex justify-between space-x-2">
              <span class="font-medium">${valueColumnName}</span>
              <span>${formattedValue} (${percentage}%)</span>
            </div>`;
          } else {
            tooltipContent += `<div class="mt-1">${formattedValue} (${percentage}%)</div>`;
          }

          // Add extra data if available
          const extraData = extraDataByName[params.name];
          if (extraData) {
            tooltipContent += "<div class=\"mt-2\">";
            Object.entries(extraData).forEach(([key, valueData]) => {
              if (Array.isArray(valueData) && valueData.length >= 2) {
                const [value, columnType] = valueData;
                tooltipContent += `<div class="flex justify-between space-x-2">
                  <span class="font-medium">${echartsEncode(key)}</span>
                  <span>${echartsEncode(formatValue(value, columnType, true))}</span>
                </div>`;
              } else {
                tooltipContent += `<div class="flex justify-between space-x-2">
                  <span class="font-medium">${echartsEncode(key)}</span>
                  <span>${echartsEncode(valueData)}</span>
                </div>`;
              }
            });
            tooltipContent += "</div>";
          }

          tooltipContent += "</div>";
          return tooltipContent;
        },
      },
      series: [series],
    };
  }, [
    data,
    isDarkMode,
    chartWidth,
    chartHeight,
    label,
    valueFormatter,
    isDonut,
  ]);

  const handleChartReady = useCallback((chart: ECharts) => {
    chartRef.current = chart;
    setChartWidth(chart.getWidth());
    setChartHeight(chart.getHeight());
  }, []);

  const handleChartResize = useCallback((chart: ECharts) => {
    setChartWidth(chart.getWidth());
    setChartHeight(chart.getHeight());
  }, []);

  return (
    <div
      className={cx("h-full w-full relative select-none overflow-hidden", className)}
      {...other}
    >
      <EChart
        className="relative h-full w-full"
        option={chartOptions}
        onChartReady={handleChartReady}
        onResize={handleChartResize}
        data-chart-id={chartId}
      />
    </div>
  );
};

PieChart.displayName = "PieChart";

export { PieChart, type PieChartProps };
