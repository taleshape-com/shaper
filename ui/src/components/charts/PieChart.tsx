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
  showLegend?: boolean;
  isDonut?: boolean;
}

const PieChart = (props: PieChartProps) => {
  const {
    data,
    valueFormatter,
    showLegend = true,
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
    const legendTopOffset = showLegend ? 35 : 0;
    const totalTopOffset = labelTopOffset + legendTopOffset;

    // Calculate center position to account for title and legend
    const availableHeight = chartHeight - totalTopOffset - chartPadding * 2;
    const centerY = totalTopOffset + chartPadding + availableHeight / 2;

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
      emphasis: {
        itemStyle: {
          shadowBlur: 10,
          shadowOffsetX: 0,
          shadowColor: "rgba(0, 0, 0, 0.5)",
        },
      },
      label: {
        show: chartWidth > 350 && data.length <= 8,
        formatter: (params: any) => {
          const percent = params.percent.toFixed(1);
          return `${params.name}: ${percent}%`;
        },
        fontFamily: chartFont,
        color: theme.textColorSecondary,
        fontSize: 12,
      },
      labelLine: {
        show: chartWidth > 350 && data.length <= 8,
      },
      animationDelay: 100,
      animationDelayUpdate: 100,
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
      legend: {
        show: showLegend,
        type: "scroll",
        orient: "horizontal",
        left: "center",
        top: labelTopOffset + chartPadding,
        textStyle: {
          color: theme.textColor,
          fontFamily: chartFont,
          fontWeight: 500,
        },
        pageButtonPosition: "end",
        pageIconColor: theme.textColorSecondary,
        pageIconInactiveColor: theme.borderColor,
        pageIcons: {
          horizontal: [
            "M10.8284 12.0007L15.7782 16.9504L14.364 18.3646L8 12.0007L14.364 5.63672L15.7782 7.05093L10.8284 12.0007Z",
            "M13.1717 12.0007L8.22192 7.05093L9.63614 5.63672L16.0001 12.0007L9.63614 18.3646L8.22192 16.9504L13.1717 12.0007Z",
          ],
        },
        pageIconSize: 12,
        pageFormatter: () => "",
      },
      series: [series],
    };
  }, [
    data,
    isDarkMode,
    chartWidth,
    chartHeight,
    label,
    showLegend,
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
