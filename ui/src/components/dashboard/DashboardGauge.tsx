// SPDX-License-Identifier: MPL-2.0

import React, { useCallback, useRef } from "react";
import { Column, GaugeCategory, Result } from "../../lib/types";
import { EChart } from "../charts/EChart";
import type { ECharts } from "echarts/core";
import { getThemeColors, getChartFont, AvailableEChartsColors, getEChartsColor, getDisplayFont } from "../../lib/chartUtils";
import { DarkModeContext } from "../../contexts/DarkModeContext";
import { formatValue } from "../../lib/render";

type DashboardGaugeProps = {
  chartId: string;
  headers: Column[];
  data: Result["sections"][0]["queries"][0]["rows"];
  gaugeCategories: GaugeCategory[];
  label?: string;
};

const barWidth = 42;
const chartPadding = 16;

const DashboardGauge: React.FC<DashboardGaugeProps> = ({
  chartId,
  headers,
  data,
  gaugeCategories,
  label,
}) => {
  const chartRef = useRef<ECharts | null>(null);
  const { isDarkMode } = React.useContext(DarkModeContext);
  const [chartSize, setChartSize] = React.useState<{ width: number, height: number }>({ width: 0, height: 0 });

  const chartOptions = React.useMemo(() => {
    const theme = getThemeColors(isDarkMode);
    const chartFont = getChartFont();
    const displayFont = getDisplayFont();

    // Find the value column
    const valueIndex = headers.findIndex(h => h.tag === "value");
    const valueHeader = headers[valueIndex];
    const value = data[0][valueIndex];

    const gaugeCategoriesWithColor = gaugeCategories.map((cat, i) => {
      if (cat.color) return cat;
      let color = theme.borderColor;
      if (i > 0) {
        const colorKey = AvailableEChartsColors[i + 1 % AvailableEChartsColors.length];
        color = getEChartsColor(colorKey, isDarkMode);
      }
      return {
        ...cat,
        color,
      };
    });

    // Calculate all unique boundary values (from and to)
    const boundaryValues = Array.from(new Set([
      ...gaugeCategoriesWithColor.map(cat => cat.from),
      ...gaugeCategoriesWithColor.map(cat => cat.to),
    ])).sort((a, b) => a - b);

    // Calculate min/max and boundaries
    const min = boundaryValues[0];
    const max = boundaryValues[boundaryValues.length - 1];

    // Color stops for axisLine
    const colorStops = gaugeCategoriesWithColor.map(cat => [
      (cat.to - min) / (max - min),
      cat.color!,
    ]) as [number, string][];

    // Helper to check if a value is a boundary (with float tolerance)
    function isBoundary (val: number) {
      return boundaryValues.some(b => Math.abs(b - val) < 1e-6);
    }

    // axisLabel formatter: only show value at boundaries
    function valueLabelFormatter (v: number) {
      return isBoundary(v) ? formatValue(v, valueHeader.type, true).toString() : "";
    }

    // Helper to calculate GCD
    function gcd (a: number, b: number): number {
      return b === 0 ? a : gcd(b, a % b);
    }

    // Calculate GCD of all differences between consecutive boundaries
    const diffs = [];
    for (let i = 1; i < boundaryValues.length; i++) {
      diffs.push(boundaryValues[i] - boundaryValues[i - 1]);
    }
    const boundaryGCD = diffs.reduce((acc, val) => gcd(acc, val));
    const splitNumber = Math.min((max - min) / boundaryGCD, 1000);

    const centerValues = diffs.map((d, i) => {
      return boundaryValues[i] + d / 2;
    });

    const labelTopOffset = label ? 20 + 15 * (Math.ceil(label.length / (0.125 * chartSize.width)) - 1) : 0;

    const radius = Math.min(chartSize.width, chartSize.height) * (chartSize.width > 540 ? 0.52 : 0.40);
    const centerPx = [
      chartSize.width / 2,
      chartSize.height / 2 + radius / 2,
    ];

    const baseSeries = {
      type: "gauge" as const,
      min,
      max,
      axisTick: {
        show: false,
      },
      splitLine: {
        show: false,
      },
      title: {
        show: false,
      },
      data: [
        {
          value: typeof value === "number" ? value : Number(value),
          name: label || valueHeader?.name || "",
        },
      ],
      startAngle: 180,
      endAngle: 0,
      center: [centerPx[0], centerPx[1] + labelTopOffset],
      radius,
    };

    // Using custom graphics to draw labels size with axisLabel we cannot control the individual alignment to ensure they don't overlap with the bar
    const graphics = (chartSize.width > 0 && chartSize.height > 0)
      ? centerValues.map((v, i) => {
        const relative = (v - min) / (max - min);
        const angle = Math.PI - (relative) * Math.PI; // 180° to 0°
        const x = centerPx[0] + (radius + 9) * Math.cos(angle);
        const y = centerPx[1] - (radius + 9) * Math.sin(angle) + labelTopOffset;
        return {
          type: "text",
          x,
          y,
          style: {
            text: gaugeCategoriesWithColor[i].label ?? "",
            fill: theme.textColorSecondary,
            font: `500 12px ${chartFont}`,
            textAlign: relative < 0.4 ? "right" : relative > 0.6 ? "left" : "center",
            textVerticalAlign: "middle",
          },
          z: 100,
          cursor: "default",
        };
      })
      : [];

    const pointerOffset = barWidth - radius;

    return {
      title: {
        text: label,
        textStyle: {
          fontSize: 16,
          lineHeight: 16,
          fontFamily: displayFont,
          fontWeight: 600,
          color: theme.textColor,
          width: chartSize.width - 10 - 2 * chartPadding,
          overflow: "break",
        },
        textAlign: "center",
        left: "50%",
        top: chartPadding,
      },
      series: [
        {
          ...baseSeries,
          splitNumber,
          // Avoids cursor:pointer on pointer hover. Need to change if we ever want to make the gauge interactive
          silent: true,
          axisLine: {
            lineStyle: {
              width: barWidth,
              color: colorStops,
            },
          },
          axisLabel: {
            distance: 28,
            color: theme.textColorSecondary,
            fontSize: 12,
            fontFamily: chartFont,
            formatter: valueLabelFormatter,
          },
          progress: {
            show: gaugeCategories.length < 2,
            width: barWidth,
            itemStyle: {
              color: theme.primaryColor,
            },
          },
          pointer: {
            show: gaugeCategories.length >= 2,
            icon: "triangle",
            length: 16,
            width: 14,
            offsetCenter: [0, pointerOffset],
            itemStyle: {
              color: theme.textColor,
            },
          },
          detail: {
            valueAnimation: false,
            fontSize: chartSize.width > 300 ? 36 : 24,
            fontFamily: chartFont,
            offsetCenter: [0, "-26%"],
            color: theme.textColor,
            fontWeight: 600,
            formatter: function (v: number) {
              return formatValue(v, valueHeader.type, true).toString();
            },
          },
        },
      ],
      graphic: graphics,
    };
  }, [
    data,
    isDarkMode,
    gaugeCategories,
    headers,
    label,
    chartSize,
  ]);

  const handleChartReady = useCallback((chart: ECharts) => {
    chartRef.current = chart;
    setChartSize({ width: chart.getWidth(), height: chart.getHeight() });
  }, []);

  const handleChartResize = useCallback((chart: ECharts) => {
    setChartSize({ width: chart.getWidth(), height: chart.getHeight() });
  }, []);

  return (
    <div className="w-full h-full relative select-none overflow-hidden">
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

export default DashboardGauge;
