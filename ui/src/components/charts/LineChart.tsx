// SPDX-License-Identifier: MPL-2.0

import React, { useEffect, useCallback, useRef } from "react";
import type { ECharts } from 'echarts/core';
import type { LineSeriesOption } from 'echarts/charts';
import {
  constructCategoryColors,
  getThemeColors,
  getChartFont,
  getDisplayFont,
} from "../../lib/chartUtils";
import { cx } from "../../lib/utils";
import { ChartHoverContext } from "../../contexts/ChartHoverContext";
import { DarkModeContext } from "../../contexts/DarkModeContext";
import { Column, isTimeType } from "../../lib/types";
import { formatValue } from "../../lib/render";
import { EChart } from "./EChart";

interface LineChartProps extends React.HTMLAttributes<HTMLDivElement> {
  chartId: string;
  label?: string;
  data: Record<string, any>[];
  extraDataByIndexAxis: Record<string, Record<string, any>>;
  index: string;
  indexType: Column['type'];
  valueType: Column['type'];
  categories: string[];
  colorsByCategory: Record<string, string>;
  valueFormatter: (value: number, shortFormat?: boolean) => string;
  indexFormatter: (value: number, shortFormat?: boolean) => string;
  showLegend?: boolean;
  xAxisLabel?: string;
  yAxisLabel?: string;
}

const chartPadding = 16;

const LineChart = (props: LineChartProps) => {
  const {
    data,
    extraDataByIndexAxis,
    categories,
    colorsByCategory,
    index,
    indexType,
    valueType,
    valueFormatter,
    indexFormatter,
    showLegend = true,
    className,
    xAxisLabel,
    yAxisLabel,
    chartId,
    label,
    ...other
  } = props;

  const chartRef = useRef<ECharts | null>(null);
  const [chartWidth, setChartWidth] = React.useState(0);
  const [chartHeight, setChartHeight] = React.useState(0);
  const hoveredChartIdRef = useRef<string | null>(null);

  const { hoveredIndex, hoveredChartId, hoveredIndexType, setHoverState } =
    React.useContext(ChartHoverContext);

  const { isDarkMode } = React.useContext(DarkModeContext);

  // Update hoveredChartId ref whenever it changes
  useEffect(() => {
    hoveredChartIdRef.current = hoveredChartId;
  }, [hoveredChartId]);

  // Memoize the chart options to prevent unnecessary re-renders
  const chartOptions = React.useMemo(() => {
    // Get computed colors for theme
    const { borderColor, textColor, textColorSecondary, referenceLineColor } = getThemeColors(isDarkMode);
    const chartFont = getChartFont();
    const displayFont = getDisplayFont();
    const categoryColors = constructCategoryColors(categories, colorsByCategory, isDarkMode);

    const isTimestampData = isTimeType(indexType) || indexType === "time" || indexType === "duration";

    // Set up chart options
    const series: LineSeriesOption[] = categories.map((category) => {
      const baseSeries: LineSeriesOption = {
        name: category,
        id: category,
        type: 'line' as const,
        data: isTimestampData
          ? data.map((item) => [item[index], item[category]])
          : data.map((item) => item[category]),
        connectNulls: true,
        symbol: 'circle',
        symbolSize: 6,
        emphasis: {
          itemStyle: {
            color: categoryColors.get(category),
            borderWidth: 0,
            shadowBlur: 0,
            shadowColor: categoryColors.get(category),
            opacity: 1,
          },
          lineStyle: {
            width: 2,
          },
        },
        lineStyle: {
          color: categoryColors.get(category),
          width: 2,
        },
        itemStyle: {
          color: categoryColors.get(category),
          borderWidth: 0,
          // always show dots when there are not too many data points and we only have a single line
          opacity: categories.length > 1 || (data.length / (chartWidth) > 0.02) ? 0 : 1,
        },
        markLine: {
          silent: true,
          symbol: 'none',
          label: {
            show: false,
          },
          data: [],
        },
      };

      // Add markLine if we're hovering on a different chart
      if (hoveredIndex != null && hoveredIndexType === indexType && hoveredChartId != null && hoveredChartId !== chartId) {
        return {
          ...baseSeries,
          markLine: {
            silent: true,
            symbol: 'none',
            label: {
              show: false,
            },
            lineStyle: {
              color: referenceLineColor,
            },
            data: [
              { xAxis: isTimestampData ? hoveredIndex : data.findIndex(item => item[index] === hoveredIndex) }
            ],
          },
        };
      }

      return baseSeries;
    });

    const numLegendItems = categories.filter(c => c.length > 0).length;
    const avgLegendCharCount = categories.reduce((acc, c) => acc + c.length, 0) / numLegendItems;
    const minLegendItemWidth = Math.max(avgLegendCharCount * 5.8, 50);
    const legendPaddingLeft = 5;
    const legendPaddingRight = 5;
    const legendItemGap = 10;
    const legendWidth = chartWidth - legendPaddingLeft - legendPaddingRight;
    const halfLegendItems = Math.ceil(numLegendItems / 2);
    const legendItemWidth = numLegendItems === 1
      ? legendWidth
      : (legendWidth - (legendItemGap * (halfLegendItems - 1))) / halfLegendItems;
    const canFitLegendItems = legendItemWidth >= minLegendItemWidth;
    const legendTopOffset = (showLegend ? (legendWidth / numLegendItems >= minLegendItemWidth ? 35 : 58) : 0);
    const labelTopOffset = label ? 36 + 15 * (Math.ceil(label.length / (0.125 * chartWidth)) - 1) : 0;
    const spaceForXaxisLabel = 10 + (xAxisLabel ? 25 : 0);
    const xData = !isTimestampData ? data.map((item) => item[index]) : undefined
    const xLabelSpace = xData && chartWidth / xData.map(x => indexFormatter(indexType === 'duration' ? new Date(x).getTime() : x, true)).join('').length;

    return {
      animation: false,
      // Quality settings for sharper rendering
      progressive: 0, // Disable progressive rendering for better quality
      progressiveThreshold: 0,
      useDirtyRect: true,
      cursor: 'default',
      title: {
        text: label,
        textStyle: {
          fontSize: 16,
          fontFamily: displayFont,
          fontWeight: 600,
          color: textColor,
          width: chartWidth - 10 - 2 * chartPadding,
          overflow: 'break',
        },
        textAlign: 'center',
        left: '50%',
        top: chartPadding,
      },
      tooltip: {
        show: true,
        trigger: 'axis',
        triggerOn: 'mousemove',
        enterable: false,
        confine: true,
        hideDelay: 200, // Increase hide delay to prevent flickering
        showDelay: 0, // Show immediately
        borderRadius: 5,
        backgroundColor: undefined,
        borderColor,
        className: 'bg-cbg dark:bg-dbg',
        textStyle: {
          fontFamily: chartFont,
          color: textColor,
        },
        formatter: (params: any) => {
          let indexValue: any;

          const axisData = params.find((item: any) => item?.axisDim === 'x');
          if (isTimestampData) {
            indexValue = axisData.value[0]; // timestamp is the first element
          } else {
            indexValue = axisData.axisValue;
          }
          const extraData = extraDataByIndexAxis[indexValue];

          const formattedIndex = indexFormatter(indexType === 'duration' ? new Date(indexValue).getTime() : indexValue);
          let tooltipContent = `<div class="text-sm font-medium">${formattedIndex}</div>`;

          if (extraData) {
            tooltipContent += `<div class="mt-2">`;
            Object.entries(extraData).forEach(([key, valueData]) => {
              if (Array.isArray(valueData) && valueData.length >= 2) {
                const [value, columnType] = valueData;
                tooltipContent += `<div class="flex justify-between space-x-2">
                  <span class="font-medium">${key}</span>
                  <span>${formatValue(value, columnType, true)}</span>
                </div>`;
              }
            });
            tooltipContent += `</div>`;
          }

          // Use a Set to track shown categories
          tooltipContent += `<div class="mt-2">`;
          params.forEach((param: any) => {
            if (param.axisDim !== 'x') {
              return; // Skip non-index axis items
            }

            let value: number;
            if (isTimestampData && Array.isArray(param.value) && param.value.length >= 2) {
              value = param.value[1] as number;
            } else {
              value = param.value as number;
            }

            // Skip categories with missing or null values
            if (value === null || value === undefined || isNaN(value)) {
              return;
            }

            const formattedValue = formatValue(value, valueType, true);
            tooltipContent += `<div class="flex items-center justify-between space-x-2">
              <div class="flex items-center space-x-2">
                <span class="inline-block size-2 rounded-sm" style="background-color: ${param.color}"></span>
                ${categories.length > 1 ? `<span>${param.seriesName}</span>` : ''}
              </div>
              <span class="font-medium">${formattedValue}</span>
            </div>`;
          });
          tooltipContent += `</div>`;

          return tooltipContent;
        },
      },
      legend: {
        show: showLegend,
        type: canFitLegendItems ? 'plain' : 'scroll',
        orient: 'horizontal',
        left: chartPadding,
        top: 7 + labelTopOffset + chartPadding,
        padding: [5, canFitLegendItems ? legendPaddingRight : 25, 5, legendPaddingLeft],
        textStyle: {
          color: textColor,
          fontFamily: chartFont,
          fontWeight: 500,
          width: canFitLegendItems ? legendItemWidth : undefined,
          overflow: 'truncate',
        },
        itemStyle: {
          opacity: 1,
          borderWidth: 0,
        },
        itemGap: legendItemGap,
        itemHeight: 8,
        itemWidth: 16,
        pageButtonPosition: 'end',
        pageButtonGap: 10,
        pageButtonItemGap: 5,
        pageIconColor: textColorSecondary,
        pageIconInactiveColor: borderColor,
        pageIcons: {
          horizontal: [
            'M10.8284 12.0007L15.7782 16.9504L14.364 18.3646L8 12.0007L14.364 5.63672L15.7782 7.05093L10.8284 12.0007Z',
            'M13.1717 12.0007L8.22192 7.05093L9.63614 5.63672L16.0001 12.0007L9.63614 18.3646L8.22192 16.9504L13.1717 12.0007Z'
          ]
        },
        pageIconSize: 12,
        pageFormatter: () => '',
        pageTextStyle: {
          fontSize: 1,
        },
        // Enable multi-row layout
        width: 'auto',
        height: categories.length > 4 ? 40 : 20, // Allow more height for multi-row
      },
      grid: {
        left: (yAxisLabel ? 45 : 15) + chartPadding,
        right: 15 + chartPadding,
        top: 10 + legendTopOffset + labelTopOffset + chartPadding,
        bottom: (xAxisLabel ? 35 : 10) + chartPadding,
        containLabel: true,
      },
      xAxis: {
        type: isTimestampData ? "time" as const : "category" as const,
        data: xData,
        show: true,
        axisLabel: {
          show: true,
          formatter: (value: any) => {
            return indexFormatter(indexType === 'duration' ? new Date(value).getTime() : value, true);
          },
          color: textColorSecondary,
          fontFamily: chartFont,
          fontSize: xLabelSpace && xLabelSpace < 15 ? 10 : 12,
          rotate: !xAxisLabel && xLabelSpace && xLabelSpace < 11 ? 45 : 0,
          padding: [4, 8, 4, 8],
          hideOverlap: true,
          showMinLabel: isTimestampData || undefined,
        },
        axisPointer: {
          type: data.length > 1 ? 'line' : 'none',
          show: true,
          triggerOn: 'mousemove',
          label: {
            show: true,
            formatter: (params: any) => {
              return indexFormatter(indexType === "number" && params.value > 1 ? Math.round(params.value) : indexType === 'duration' ? new Date(params.value).getTime() : params.value);
            },
            fontFamily: chartFont,
            margin: 5,
          }
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
        nameLocation: 'middle',
        nameGap: 40,
        nameTextStyle: {
          color: textColor,
          fontFamily: chartFont,
          fontWeight: 500,
          fontSize: 14,
        },
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
          type: 'line',
          show: data.length > 1,
          triggerOn: 'mousemove',
          triggerEmphasis: false,
          label: {
            show: true,
            formatter: (params: any) => {
              return valueFormatter(valueType === "number" && params.value > 1 ? Math.round(params.value) : params.value);
            },
            fontFamily: chartFont,
            margin: 10,
          }
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
      graphic: [
        // We use a graphic element to display the y-axis label instead of the axis name
        // since the name can overlap with the axis labels.
        // See https://github.com/apache/echarts/issues/12415#issuecomment-2285226567
        {
          type: "text",
          rotation: Math.PI / 2,
          y: (chartHeight + labelTopOffset + legendTopOffset - spaceForXaxisLabel) / 2,
          x: 5 + chartPadding,
          style: {
            text: yAxisLabel,
            font: `500 14px ${chartFont}`,
            fill: textColor,
            width: chartHeight,
            textAlign: "center",
          }
        },
      ],
      series,
    };
  }, [
    data,
    categories,
    colorsByCategory,
    index,
    indexType,
    valueType,
    valueFormatter,
    indexFormatter,
    showLegend,
    xAxisLabel,
    yAxisLabel,
    extraDataByIndexAxis,
    hoveredIndex,
    hoveredChartId,
    hoveredIndexType,
    chartId,
    isDarkMode,
    chartWidth,
    chartHeight,
    label,
  ]);

  // Event handlers for the EChart component
  const chartEvents = React.useMemo(() => {
    return {
      // Add tooltip event handler
      showTip: (params: any) => {
        // Handle both dataIndex and axisValue approaches
        let indexValue: any;

        if (params.dataIndex !== undefined && params.seriesIndex !== undefined) {
          // Use dataIndex if available
          const dataIndex = params.dataIndex;
          if (dataIndex >= 0 && dataIndex < data.length) {
            indexValue = data[dataIndex][index];
          }
        } else if (params.axisValue !== undefined) {
          // Use axisValue as fallback
          indexValue = params.axisValue;
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
  }, [indexType, data, index, chartId, setHoverState]);

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
        className="absolute inset-0"
        option={chartOptions}
        events={chartEvents}
        onChartReady={handleChartReady}
        onResize={handleChartResize}
        data-chart-id={chartId}
      />
    </div>
  );
};

LineChart.displayName = "LineChart";

export { LineChart, type LineChartProps };