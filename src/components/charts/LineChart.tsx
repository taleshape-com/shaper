import React, { useEffect, useCallback, useRef } from "react";
import * as echarts from "echarts";
import {
  AvailableEChartsColors,
  constructEChartsCategoryColors,
  getEChartsColor,
  isDarkMode,
  getThemeColors,
} from "../../lib/chartUtils";
import { cx } from "../../lib/utils";
import { ChartHoverContext } from "../../contexts/ChartHoverContext";
import { Column } from "../../lib/dashboard";
import { formatValue } from "../../lib/render";
import { EChart } from "./EChart";
import { ChartDownloadButton } from "./ChartDownloadButton";

interface LineChartProps extends React.HTMLAttributes<HTMLDivElement> {
  chartId: string;
  data: Record<string, any>[];
  extraDataByIndexAxis: Record<string, Record<string, any>>;
  index: string;
  indexType: Column['type'];
  valueType: Column['type'];
  categories: string[];
  valueFormatter: (value: number) => string;
  indexFormatter: (value: number) => string;
  showLegend?: boolean;
  xAxisLabel?: string;
  yAxisLabel?: string;
  min?: number;
  max?: number;
  label?: string;
}

const LineChart = React.forwardRef<HTMLDivElement, LineChartProps>(
  (props, forwardedRef) => {
    const {
      data,
      extraDataByIndexAxis,
      categories,
      index,
      indexType,
      valueType,
      valueFormatter = (value: number) => value.toString(),
      indexFormatter = (value: number) => value.toString(),
      showLegend = true,
      className,
      xAxisLabel,
      yAxisLabel,
      min,
      max,
      chartId,
      label,
      ...other
    } = props;

    const chartRef = useRef<echarts.ECharts | null>(null);
    const hoveredChartIdRef = useRef<string | null>(null);

    const { hoveredIndex, hoveredChartId, hoveredIndexType, setHoverState } =
      React.useContext(ChartHoverContext);

    const categoryColors = constructEChartsCategoryColors(categories, AvailableEChartsColors);

    // Update hoveredChartId ref whenever it changes
    useEffect(() => {
      hoveredChartIdRef.current = hoveredChartId;
    }, [hoveredChartId]);

    // Memoize the chart options to prevent unnecessary re-renders
    const chartOptions = React.useMemo((): echarts.EChartsOption => {
      // Get computed colors for theme
      const { backgroundColor, borderColor, textColor, textColorSecondary, referenceLineColor } = getThemeColors();
      const isDark = isDarkMode();

      // Check if we're dealing with timestamps
      const isTimestampData = indexType === "date" || indexType === "timestamp" || indexType === "hour" || indexType === "month" || indexType === "year" || indexType === "time";

      // Set up chart options
      const series: echarts.LineSeriesOption[] = categories.map((category) => {
        const baseSeries: echarts.LineSeriesOption = {
          name: category,
          type: 'line' as const,
          data: isTimestampData
            ? data.map((item) => [item[index], item[category]])
            : data.map((item) => item[category]),
          connectNulls: true,
          symbol: 'circle',
          symbolSize: (params: any) => {
            // Show symbol only when hovering
            return params.dataIndex === hoveredIndex ? 6 : 0;
          },
          emphasis: {
            itemStyle: {
              color: getEChartsColor(categoryColors.get(category) || 'primary', isDark),
              borderColor: backgroundColor,
              borderWidth: 2,
              shadowBlur: 0,
              shadowColor: getEChartsColor(categoryColors.get(category) || 'primary', isDark),
            },
            lineStyle: {
              width: 2,
            },
          },
          lineStyle: {
            color: getEChartsColor(categoryColors.get(category) || 'primary', isDark),
            width: 2,
          },
          itemStyle: {
            color: getEChartsColor(categoryColors.get(category) || 'primary', isDark),
            borderColor: backgroundColor,
            borderWidth: 2,
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
                { xAxis: hoveredIndex },
              ],
            },
          };
        }

        return baseSeries;
      });

      // For category axis, we need to use the index values as category names
      const xAxisData = data.map((item) => item[index]);

      return {
        animation: false,
        // Quality settings for sharper rendering
        progressive: 0, // Disable progressive rendering for better quality
        progressiveThreshold: 0,
        // Anti-aliasing and rendering quality
        // renderer: 'canvas',
        useDirtyRect: true,
        tooltip: {
          show: true,
          trigger: 'axis',
          triggerOn: 'mousemove',
          enterable: false,
          confine: true,
          hideDelay: 200, // Increase hide delay to prevent flickering
          showDelay: 0, // Show immediately
          borderRadius: 5,
          formatter: (params: any) => {
            let indexValue: any;
            let extraData: any;

            if (isTimestampData) {
              // For time axis, the first param contains the timestamp
              indexValue = params[0].value[0]; // timestamp is the first element
              const dataIndex = data.findIndex(item => item[index] === indexValue);
              extraData = dataIndex >= 0 ? extraDataByIndexAxis[data[dataIndex][index]] : undefined;
            } else {
              // For category axis, use the original logic
              indexValue = params[0].axisValue;
              extraData = extraDataByIndexAxis[indexValue];
            }

            const formattedIndex = indexFormatter(indexValue);

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

            tooltipContent += `<div class="mt-2">`;
            params.forEach((param: any) => {
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

              const formattedValue = formatValue(value, 'number', true);
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
          backgroundColor,
          borderColor,
          textStyle: {
            color: textColor,
          },
        },
        legend: {
          show: showLegend,
          type: 'scroll',
          orient: 'horizontal',
          left: 10,
          top: '0',
          textStyle: {
            color: textColor,
          },
        },
        grid: {
          left: yAxisLabel ? 45 : 15,
          right: 10,
          top: showLegend ? 40 : 10,
          bottom: xAxisLabel ? 35 : 10,
          containLabel: true,
        },
        xAxis: {
          type: isTimestampData ? "time" as const : "category" as const,
          data: !isTimestampData ? xAxisData : undefined,
          show: true,
          axisLabel: {
            show: true,
            formatter: (value: any) => {
              return indexFormatter(value);
            },
            color: textColorSecondary,
            rotate: 0,
            margin: 16,
            padding: [4, 8, 4, 8],
            interval: (index: number) => {
              // Always show first and last labels
              if (index === 0 || index === data.length - 1) {
                return true;
              }
              // For intermediate labels, show every nth label
              const step = Math.max(1, Math.floor(data.length / 3));
              return index % step === 0;
            },
            hideOverlap: true,
          },
          axisPointer: {
            type: 'line',
            show: valueType === 'number' || valueType === 'duration' || valueType === 'percent',
            triggerOn: 'mousemove',
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
          name: xAxisLabel,
          nameLocation: 'middle',
          nameGap: 45,
          nameTextStyle: {
            color: textColor,
          },
          min,
          max,
        },
        yAxis: {
          type: "value" as const,
          show: true,
          axisLabel: {
            show: true,
            formatter: (value: any) => {
              return valueFormatter(value);
            },
            color: textColorSecondary,
            margin: 16,
            padding: [4, 8, 4, 8],
            hideOverlap: true,
          },
          axisPointer: {
            type: 'line',
            show: valueType === 'number' || valueType === 'duration' || valueType === 'percent',
            triggerOn: 'mousemove',
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
          name: yAxisLabel,
          nameLocation: 'middle',
          nameGap: 60,
          nameTextStyle: {
            color: textColor,
          },
        },
        series,
      };
    }, [
      data,
      categories,
      index,
      indexType,
      valueType,
      valueFormatter,
      indexFormatter,
      showLegend,
      xAxisLabel,
      yAxisLabel,
      min,
      max,
      categoryColors,
      extraDataByIndexAxis,
      hoveredIndex,
      hoveredChartId,
      hoveredIndexType,
      chartId,
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

    // Handle chart instance reference
    const handleChartReady = useCallback((chart: echarts.ECharts) => {
      chartRef.current = chart;
    }, []);

    return (
      <div
        ref={forwardedRef}
        className={cx("h-80 w-full relative group", className)}
        {...other}
      >
        <EChart
          className="absolute inset-0"
          option={chartOptions}
          events={chartEvents}
          onChartReady={handleChartReady}
        />
        <ChartDownloadButton
          chartRef={chartRef}
          chartId={chartId}
          label={label}
          showLegend={showLegend}
        />
      </div>
    );
  }
);

LineChart.displayName = "LineChart";

export { LineChart, type LineChartProps };