// SPDX-License-Identifier: MPL-2.0

import React, { useCallback, useRef } from "react";
import type { ECharts } from "echarts/core";
import {
  getThemeColors,
  getChartFont,
  getDisplayFont,
} from "../../lib/chartUtils";
import { cx } from "../../lib/utils";
import { DarkModeContext } from "../../contexts/DarkModeContext";
import { echartsEncode } from "../../lib/render";
import { EChart } from "./EChart";

interface HeatmapChartProps extends React.HTMLAttributes<HTMLDivElement> {
  chartId: string;
  label?: string;
  data: [string, number][];
  valueFormatter: (value: number) => string;
  valueColumnName?: string;
}

const chartPadding = 16;

const HeatmapChart = (props: HeatmapChartProps) => {
  const {
    data,
    valueFormatter,
    valueColumnName,
    className,
    chartId,
    label,
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

    // Determine year range from data
    const years = new Set<number>();
    let minValue = Infinity;
    let maxValue = -Infinity;
    data.forEach(([date, value]) => {
      const y = Number.parseInt(date.slice(0, 4), 10);
      if (Number.isFinite(y)) {
        years.add(y);
      }
      if (typeof value === "number" && Number.isFinite(value)) {
        if (value < minValue) minValue = value;
        if (value > maxValue) maxValue = value;
      }
    });

    if (!Number.isFinite(minValue)) {
      minValue = 0;
    }
    if (!Number.isFinite(maxValue)) {
      maxValue = 0;
    }
    // visualMap requires min < max
    if (minValue === maxValue) {
      if (maxValue === 0) {
        maxValue = 1;
      } else {
        minValue = Math.min(0, minValue);
      }
    }

    const sortedYears = Array.from(years).sort((a, b) => a - b);
    const currentYear = new Date().getUTCFullYear();
    const rangeYears = sortedYears.length > 0 ? sortedYears : [currentYear];

    const labelTopOffset = label
      ? 40 + 15 * (Math.ceil(label.length / (0.125 * chartWidth)) - 1)
      : 25;

    // Layout calendars stacked vertically when multiple years are present
    const calendars = rangeYears.map((year, idx) => {
      const yearGap = 18;
      const yearHeight = Math.max(
        80,
        (chartHeight - labelTopOffset - chartPadding * 2 - 50 - (rangeYears.length - 1) * yearGap) /
          rangeYears.length,
      );
      const top = labelTopOffset + chartPadding + idx * (yearHeight + yearGap);
      return {
        range: year.toString(),
        top,
        left: 50,
        right: 20,
        cellSize: ["auto", Math.max(10, Math.min(20, yearHeight / 9))],
        orient: "horizontal",
        splitLine: {
          show: true,
          lineStyle: {
            color: theme.borderColor,
            width: 1,
            type: "solid",
          },
        },
        itemStyle: {
          color: theme.backgroundColorSecondary,
          borderColor: theme.borderColor,
          borderWidth: 0.5,
        },
        yearLabel: {
          show: rangeYears.length > 1,
          color: theme.textColorSecondary,
          fontFamily: chartFont,
          fontSize: 12,
        },
        monthLabel: {
          color: theme.textColorSecondary,
          fontFamily: chartFont,
          fontSize: 11,
        },
        dayLabel: {
          color: theme.textColorSecondary,
          fontFamily: chartFont,
          fontSize: 10,
          firstDay: 1,
        },
      };
    });

    const series = rangeYears.map((year, idx) => ({
      type: "heatmap" as const,
      coordinateSystem: "calendar" as const,
      calendarIndex: idx,
      data: data.filter(([d]) => d.startsWith(year.toString())),
    }));

    return {
      title: [
        {
          text: label,
          textStyle: {
            fontSize: 15,
            lineHeight: 15,
            fontFamily: displayFont,
            fontWeight: 600,
            color: theme.textColor,
            width: chartWidth - 10 - 2 * chartPadding,
            overflow: "break",
          },
          left: "center",
          top: chartPadding,
        },
      ],
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
          if (!params.value || !Array.isArray(params.value)) {
            return "";
          }
          const [date, value] = params.value;
          const formattedValue = echartsEncode(
            valueFormatter(typeof value === "number" ? value : Number(value) || 0),
          );
          let html = `<div class="text-sm"><div class="font-medium">${echartsEncode(date)}</div>`;
          if (valueColumnName) {
            html += `<div class="mt-1 flex justify-between space-x-2"><span class="font-medium">${echartsEncode(valueColumnName)}</span><span>${formattedValue}</span></div>`;
          } else {
            html += `<div class="mt-1">${formattedValue}</div>`;
          }
          html += "</div>";
          return html;
        },
      },
      visualMap: {
        min: minValue,
        max: maxValue,
        calculable: false,
        orient: "horizontal",
        left: "center",
        bottom: 0,
        itemWidth: 12,
        itemHeight: 120,
        textStyle: {
          color: theme.textColorSecondary,
          fontFamily: chartFont,
          fontSize: 11,
        },
        inRange: {
          color: isDarkMode
            ? ["#0e4429", "#006d32", "#26a641", "#39d353"]
            : ["#ebedf0", "#9be9a8", "#40c463", "#30a14e", "#216e39"],
        },
      },
      calendar: calendars,
      series,
    };
  }, [data, isDarkMode, chartWidth, chartHeight, label, valueFormatter, valueColumnName]);

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

HeatmapChart.displayName = "HeatmapChart";

export { HeatmapChart, type HeatmapChartProps };
