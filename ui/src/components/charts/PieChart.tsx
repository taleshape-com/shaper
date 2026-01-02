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
import { echartsEncode } from "../../lib/render";
import { EChart } from "./EChart";

const chartPadding = 16;

interface PieChartProps extends React.HTMLAttributes<HTMLDivElement> {
  chartId: string;
  label?: string;
  data: { name: string; value: number; color?: string }[];
  valueType: Column["type"];
  valueFormatter: (value: number) => string;
  isDonut?: boolean;
}

const PieChart = (props: PieChartProps) => {
  const {
    data,
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
      ? 36 + 15 * (Math.ceil(label.length / (0.125 * chartWidth)) - 1)
      : 0;

    // Calculate center position to account for title
    const availableHeight = chartHeight - labelTopOffset - chartPadding * 2;
    const centerY = labelTopOffset + chartPadding + availableHeight / 2;

    const series: PieSeriesOption = {
      type: "pie",
      radius: isDonut ? ["40%", "70%"] : "70%",
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
      },
      labelLine: {
        show: false,
        length: 12,
        length2: 0,
      },
      animationDelay: 100,
      animationDelayUpdate: 100,
      cursor: "crosshair",
    };

    return {
      title: {
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
          return `<div class="text-sm">
            <div class="flex items-center space-x-2">
              <span class="inline-block size-2 rounded-sm" style="background-color: ${echartsEncode(params.color)}"></span>
              <span class="font-medium">${echartsEncode(params.name)}</span>
            </div>
            <div class="mt-1">${echartsEncode(valueFormatter(params.value))} (${percentage}%)</div>
          </div>`;
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
