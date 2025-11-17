// SPDX-License-Identifier: MPL-2.0

import React, { useEffect, useCallback, useRef } from "react";
import type { ECharts } from "echarts/core";
import type {
  BoxplotSeriesOption,
  ScatterSeriesOption,
  LineSeriesOption,
} from "echarts/charts";
import { getThemeColors, getChartFont, getDisplayFont } from "../../lib/chartUtils";
import { cx } from "../../lib/utils";
import { ChartHoverContext } from "../../contexts/ChartHoverContext";
import { DarkModeContext } from "../../contexts/DarkModeContext";
import { Column, isDatableType, MarkLine } from "../../lib/types";
import { formatValue } from "../../lib/render";
import { translate } from "../../lib/translate";
import { EChart } from "./EChart";

interface BoxplotProps extends React.HTMLAttributes<HTMLDivElement> {
  chartId: string;
  label?: string;
  data: [number, number, number, number, number][];
  outliers: [number, number, Record<string, string> | null | undefined][];
  xData: string[];
  extraDataByIndexAxis: Record<string, Record<string, any>>;
  indexType: Column["type"];
  colorByIndex: Map<number, string>;
  valueFormatter: (value: number, shortFormat?: boolean | number) => string;
  indexFormatter: (value: number | string, shortFormat?: boolean | number) => string;
  xAxisLabel?: string;
  yAxisLabel?: string;
  markLines?: MarkLine[];
}

const chartPadding = 16;

const valueKeys = [
  "min" as const,
  "Q1" as const,
  "median" as const,
  "Q3" as const,
  "max" as const,
];

const Boxplot = (props: BoxplotProps) => {
  const {
    data,
    outliers,
    xData,
    extraDataByIndexAxis,
    colorByIndex,
    indexType,
    valueFormatter,
    indexFormatter,
    className,
    xAxisLabel,
    yAxisLabel,
    chartId,
    label,
    markLines,
    ...other
  } = props;

  const chartRef = useRef<ECharts | null>(null);
  const [chartWidth, setChartWidth] = React.useState(450);
  const [chartHeight, setChartHeight] = React.useState(300);
  const hoveredChartIdRef = useRef<string | null>(null);

  const { hoveredIndex, hoveredChartId, hoveredIndexType, setHoverState } =
    React.useContext(ChartHoverContext);

  const { isDarkMode } = React.useContext(DarkModeContext);
  const [isHovering, setIsHovering] = React.useState<null | string | number>(null);

  // Update hoveredChartId ref whenever it changes
  useEffect(() => {
    hoveredChartIdRef.current = hoveredChartId;
  }, [hoveredChartId]);

  // Memoize the chart options to prevent unnecessary re-renders
  const chartOptions = React.useMemo(() => {
    // Get computed colors for theme
    const { primaryColor, borderColor, backgroundColor, textColor, textColorSecondary, referenceLineColor, backgroundColorSecondary } = getThemeColors(isDarkMode);
    const chartFont = getChartFont();
    const displayFont = getDisplayFont();

    // Set up chart options
    const series: (BoxplotSeriesOption | ScatterSeriesOption | LineSeriesOption)[] = [
      {
        name: "boxplot",
        id: "boxplot",
        type: "boxplot",
        data,
        colorBy: "data",
        // This helps hides marklines
        zlevel: 1,
        emphasis: {
          itemStyle: {
            shadowBlur: 0,
            color: backgroundColorSecondary,
            borderWidth: 1.75,
          },
        },
        itemStyle: {
          //borderColor: primaryColor,
          color: backgroundColorSecondary,
          borderWidth: 1.25,
          // This hides marklines
          shadowOffsetX: 1,
          shadowColor: backgroundColorSecondary,
        },
      },
      {
        name: "outliers",
        id: "outliers",
        type: "scatter",
        data: outliers,
        zlevel: 1,
        symbolSize: 8,
        colorBy: "data",
        itemStyle: {
          //color: primaryColor,
          opacity: 0.7,
        },
        emphasis: {
          scale: 1.4,
          itemStyle: {
            opacity: 1,
            borderWidth: 1,
            borderColor: backgroundColor,
          },
        },
        cursor: "crosshair",
      } as ScatterSeriesOption,
    ];

    if (markLines) {
      let foundEventLine = false;
      let multiEventLines = false;
      for (const m of markLines) {
        if (!m.isYAxis) {
          if (foundEventLine) {
            multiEventLines = true;
            break;
          }
          foundEventLine = true;
        }
      }
      series.push({
        type: "line",
        markLine: {
          silent: true,
          symbol: "none",
          data: markLines.map(m => {
            const v = m.isYAxis ? m.value
              : isDatableType(indexType)
                ? new Date(m.value).toUTCString()
                : m.value.toString();
            return {
              xAxis: m.isYAxis ? undefined : v as number | string,
              yAxis: m.isYAxis ? v as number : undefined,
              symbol: m.isYAxis ? "none" : "circle",
              symbolSize: 6.5,
              lineStyle: {
                color: textColorSecondary,
                type: "dashed",
                width: m.isYAxis ? 1.2 : 1,
                opacity: m.isYAxis ? 0.5 : 0.8,
                shadowColor: backgroundColorSecondary,
              },
              label: {
                formatter: m.label,
                position: m.isYAxis ? "insideStartTop" : multiEventLines ? "insideEnd" : "end",
                distance: m.isYAxis ? 1.5 : multiEventLines ? 5 : 3.8,
                color: textColorSecondary,
                fontFamily: chartFont,
                fontWeight: 500,
                fontSize: 11,
                opacity: m.isYAxis ? 0.5 : 0.8,
                width: m.isYAxis ? chartWidth / 3 : chartHeight / 2,
                overflow: "truncate",
                textBorderColor: backgroundColorSecondary,
                textBorderWidth: !m.isYAxis && multiEventLines ? 2 : 0,
              },
            };
          }),
        },
      });
    }

    const labelTopOffset = label ? 36 + 15 * (Math.ceil(label.length / (0.125 * chartWidth)) - 1) : 0;
    const spaceForXaxisLabel = 10 + (xAxisLabel ? 25 : 0);
    const xSpace = (chartWidth - 2 * chartPadding + (yAxisLabel ? 50 : 30));
    const shortenLabel = xData ? (xSpace / xData.length) * (0.10 + (0.00004 * xSpace)) : true;

    return {
      animation: false,
      color: xData.map((_, i) => colorByIndex.get(i) || primaryColor),
      title: {
        text: label,
        textStyle: {
          fontSize: 16,
          lineHeight: 16,
          fontFamily: displayFont,
          fontWeight: 600,
          color: textColor,
          width: chartWidth - 10 - 2 * chartPadding,
          overflow: "break",
        },
        textAlign: "center",
        left: "50%",
        top: chartPadding,
      },
      tooltip: {
        show: true,
        trigger: "item",
        enterable: false,
        confine: true,
        hideDelay: 200, // Increase hide delay to prevent flickering
        showDelay: 0, // Show immediately
        borderRadius: 5,
        backgroundColor: undefined,
        borderColor,
        className: "bg-cbg dark:bg-dbg",
        textStyle: {
          fontFamily: chartFont,
          color: textColor,
        },
        formatter: (params: any) => {
          const param = Array.isArray(params) ? params.find((item: any) => item?.seriesId === "boxplot" && item?.axisDim === "x") : params;
          if (!param) {
            return;
          }
          let tooltipContent = "";

          if (param.seriesId === "outliers") {
            const values = param.value as number[];
            if (values === null || values === undefined || !Array.isArray(values)) {
              return;
            }
            const formattedValue = valueFormatter(values[1], true);
            tooltipContent += `<div class="flex items-center space-x-2">
                <span class="inline-block size-3 rounded-full bg-cthree dark:bg-dthree"></span>
                <span class="text-sm font-medium">${formattedValue}</span>
              </div>`;
            const extraData = Object.entries(values[2]);
            if (extraData.length) {
              tooltipContent += "<div class=\"mt-2\">";
              extraData.forEach(([key, value]) => {
                tooltipContent += `<div class="flex justify-between space-x-2">
                  <span class="font-medium">${key}</span>
                  <span>${formatValue(value, "string", true)}</span>
                </div>`;
              });
              tooltipContent += "</div>";
            }
            return tooltipContent;
          }

          const indexValue = param.name;
          const formattedIndex = indexFormatter(decodeIndexValue(indexValue, indexType));
          tooltipContent += `<div class="text-sm font-medium">${formattedIndex}</div>`;

          const extraData = extraDataByIndexAxis[indexValue];
          if (extraData) {
            tooltipContent += "<div class=\"mt-2\">";
            Object.entries(extraData).forEach(([key, valueData]) => {
              if (Array.isArray(valueData) && valueData.length >= 2) {
                const [value, columnType] = valueData;
                tooltipContent += `<div class="flex justify-between space-x-2">
                  <span class="font-medium">${key}</span>
                  <span>${formatValue(value, columnType, true)}</span>
                </div>`;
              }
            });
            tooltipContent += "</div>";
          }

          tooltipContent += "<div class=\"mt-2\">";
          const values = param.value as number[];

          if (values === null || values === undefined || !Array.isArray(values)) {
            return;
          }

          // Skip first since it's the x-index
          for (let i = 1; i < values.length; i++) {
            const formattedValue = valueFormatter(values[i], true);
            const key = translate(valueKeys[i - 1]);
            tooltipContent += `<div class="flex items-center justify-between space-x-2">
                  <span class="font-medium">${key}</span>
                  <span>${formattedValue}</span>
            </div>`;
          }
          tooltipContent += "</div>";
          return tooltipContent;
        },
      },
      legend: {
        show: false,
      },
      grid: {
        left: (yAxisLabel ? 45 : 15) + chartPadding,
        right: 15 + chartPadding,
        top: 10 + labelTopOffset + chartPadding,
        bottom: (xAxisLabel ? 35 : 10) + chartPadding,
        containLabel: true,
      },
      xAxis: {
        type: "category",
        data: xData,
        show: true,
        axisLabel: {
          show: true,
          formatter: (value: string) => {
            return indexFormatter(decodeIndexValue(value, indexType), shortenLabel);
          },
          color: textColorSecondary,
          fontFamily: chartFont,
          fontSize: 12,
          rotate: !xAxisLabel && typeof shortenLabel === "number" && shortenLabel <= 12 ? 45 : 0,
          padding: [4, 8, 4, 8],
          hideOverlap: true,
        },
        axisPointer: {
          type: "shadow",
          show: true,
          triggerOn: "mousemove",
          triggerEmphasis: false,
          lineStyle: {
            color: referenceLineColor,
            type: "dashed",
            width: 0.8,
          },
          label: {
            show: true,
            formatter: (params: any) => {
              return indexFormatter(decodeIndexValue(params.value, indexType), shortenLabel);
            },
            fontFamily: chartFont,
            margin: 5,
          },
        },
        axisLine: {
          show: false,
        },
        axisTick: {
          show: false,
        },
        splitLine: {
          show: false,
        },
        name: xAxisLabel,
        nameLocation: "middle",
        nameGap: 40,
        nameTextStyle: {
          color: textColor,
          fontFamily: chartFont,
          fontWeight: 500,
          fontSize: 14,
        },
        jitter: chartWidth / xData.length * 0.95,
      },
      yAxis: {
        type: "value" as const,
        show: true,
        axisLabel: {
          show: true,
          formatter: (value: any) => {
            return valueFormatter(value, true);
          },
          color: textColorSecondary,
          fontFamily: chartFont,
          padding: [4, 8, 4, 8],
          hideOverlap: true,
        },
        axisPointer: {
          type: "none",
          show: data.length > 1,
          triggerOn: "mousemove",
          triggerEmphasis: false,
          label: {
            show: true,
            formatter: (params: any) => {
              return valueFormatter(params.value > 1 ? Math.round(params.value) : params.value, true);
            },
            fontFamily: chartFont,
            margin: 10,
          },
        },
        axisLine: {
          show: false,
        },
        axisTick: {
          show: false,
        },
        splitLine: {
          show: true,
          lineStyle: {
            color: borderColor,
          },
        },
      },
      // We use a graphic element to display the y-axis label instead of the axis name
      // since the name can overlap with the axis labels.
      // See https://github.com/apache/echarts/issues/12415#issuecomment-2285226567
      graphic: {
        type: "text",
        rotation: Math.PI / 2,
        y: (chartHeight + labelTopOffset - spaceForXaxisLabel) / 2,
        x: 5 + chartPadding,
        cursor: "default",
        style: {
          text: yAxisLabel,
          font: `500 14px ${chartFont}`,
          fill: textColor,
          width: chartHeight,
          textAlign: "center",
        },
      },
      series,
    };
  }, [
    data,
    xData,
    outliers,
    colorByIndex,
    indexType,
    valueFormatter,
    indexFormatter,
    xAxisLabel,
    yAxisLabel,
    extraDataByIndexAxis,
    isDarkMode,
    chartWidth,
    chartHeight,
    label,
    markLines,
  ]);

  useEffect(() => {
    if (hoveredIndex != null && hoveredIndexType === indexType && hoveredChartId != null && hoveredChartId !== chartId) {
      const strIndex = isDatableType(indexType)
        ? new Date(hoveredIndex as number).toUTCString()
        : hoveredIndex.toString();
      setIsHovering(strIndex);
    } else {
      setIsHovering(null);
    }
  }, [
    chartId,
    indexType,
    hoveredIndex,
    hoveredIndexType,
    hoveredChartId,
    setIsHovering,
  ]);

  useEffect(() => {
    const chart = chartRef.current;
    if (!chart) {
      return;
    }
    const { referenceLineColor } = getThemeColors(isDarkMode);
    const series: LineSeriesOption[] = [{
      id: "shaper-hover-reference-line",
      type: "line" as const,
      markLine: {
        silent: true,
        symbol: "none",
        label: {
          show: false,
        },
        lineStyle: {
          color: referenceLineColor,
          type: "dashed",
          width: 0.8,
        },
        data: isHovering != null ? [{ xAxis: isHovering }] : [],
      },
    }];
    chart.setOption({ series }, { lazyUpdate: true });
  }, [indexType, isDarkMode, isHovering]);

  // Event handlers for the EChart component
  const chartEvents = React.useMemo(() => {
    return {
      // Add tooltip event handler
      showTip: (params: any) => {
        let indexValue: any;
        // seriesIndex 0=box 1=outlier
        if (params.seriesIndex === 0) {
          indexValue = decodeIndexValue(xData[params.dataIndex], indexType);
        } else if (params.seriesIndex === 1) {
          const i = (outliers[params.dataIndex] ?? [])[0];
          if (i !== undefined) {
            indexValue = decodeIndexValue(xData[i], indexType);
          }
        }
        if (indexValue !== undefined) {
          setHoverState(indexValue, chartId, indexType);
        }
      },
      // Also handle tooltip hide to clear hover state
      hideTip: () => {
        if (hoveredChartIdRef.current === chartId) {
          setHoverState(null, null, null);
        }
      },
    };
  }, [indexType, xData, chartId, outliers, setHoverState]);

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
        events={chartEvents}
        onChartReady={handleChartReady}
        onResize={handleChartResize}
        data-chart-id={chartId}
      />
    </div>
  );
};

function decodeIndexValue (v: string | number, indexType: Column["type"]): string | number {
  if (isDatableType(indexType)) {
    return new Date(v).getTime();
  }
  if (indexType === "number") {
    if (typeof v === "number") {
      return v;
    }
    return parseFloat(v);
  }
  return v;
}

Boxplot.displayName = "Boxplot";

export { Boxplot, type BoxplotProps };
