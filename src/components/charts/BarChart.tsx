import React, { useEffect, useCallback } from "react";
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
import { translate } from "../../lib/translate";
import { EChart } from "./EChart";
import { ChartDownloadButton } from "./ChartDownloadButton";

interface BarChartProps extends React.HTMLAttributes<HTMLDivElement> {
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
  layout: "vertical" | "horizontal";
  type: "default" | "stacked";
  min?: number;
  max?: number;
  label?: string;
}

const BarChart = React.forwardRef<HTMLDivElement, BarChartProps>(
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
      layout,
      type,
      min,
      max,
      chartId,
      label,
      ...other
    } = props;

    const chartRef = React.useRef<echarts.ECharts | null>(null);

    const { hoveredIndex, hoveredChartId, hoveredIndexType, setHoverState } =
      React.useContext(ChartHoverContext);

    const categoryColors = constructEChartsCategoryColors(categories, AvailableEChartsColors);

    // Memoize the chart options to prevent unnecessary re-renders
    const chartOptions = React.useMemo((): echarts.EChartsOption => {
      // Get computed colors for theme
      const { backgroundColor, borderColor, textColor, textColorSecondary } = getThemeColors();
      const isDark = isDarkMode();

      // Check if we're dealing with timestamps
      const isTimestampData = indexType === "date" || indexType === "timestamp" || indexType === "hour" || indexType === "month" || indexType === "year" || indexType === "time";

      // Set up chart options
      const series: echarts.BarSeriesOption[] = categories.map((category) => {
        if (isTimestampData) {
          // For time axis, we need to provide data as [timestamp, value] pairs for both layouts
          return {
            name: category,
            type: 'bar',
            barGap: '3%',
            stack: type === "stacked" ? "stack" : undefined,
            data: data.map((item) => [item[index], item[category]]),
            itemStyle: {
              color: getEChartsColor(categoryColors.get(category) || 'primary', isDark),
            },
            emphasis: {
              itemStyle: {
                color: getEChartsColor(categoryColors.get(category) || 'primary', isDark),
                opacity: 0.8,
              },
            },
          };
        } else {
          // For category axis, use the original format
          return {
            name: category,
            type: 'bar',
            barGap: '3%',
            stack: type === "stacked" ? "stack" : undefined,
            data: data.map((item) => item[category]),
            itemStyle: {
              color: getEChartsColor(categoryColors.get(category) || 'primary', isDark),
            },
            emphasis: {
              itemStyle: {
                color: getEChartsColor(categoryColors.get(category) || 'primary', isDark),
                opacity: 0.8,
              },
            },
          };
        }
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
              // For time axis, the first param contains the timestamp for both layouts
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

            if (type === "stacked" && (valueType === "number" || valueType === "duration")) {
              const total = params.reduce((sum: number, item: any) => {
                let value: number;
                if (isTimestampData && Array.isArray(item.value) && item.value.length >= 2) {
                  value = item.value[1] as number;
                } else {
                  value = item.value as number;
                }

                // Only include valid values in the total
                if (value === null || value === undefined || isNaN(value)) {
                  return sum;
                }

                return sum + value;
              }, 0);
              tooltipContent += `<div class="flex justify-between space-x-2 mt-2">
                <span class="font-medium">${translate('Total')}</span>
                <span>${formatValue(total, valueType, true)}</span>
              </div>`;
            }

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
          type: layout === "horizontal" ? (isTimestampData ? "time" as const : "category" as const) : "value" as const,
          data: layout === "horizontal" && !isTimestampData ? xAxisData : undefined,
          show: true,
          axisLabel: {
            show: true, // Always show labels
            formatter: (value: any) => {
              if (valueType === "percent" && layout === "vertical") {
                return `${(value * 100).toFixed(0)}%`;
              }
              // For timestamps, format the value properly
              return indexFormatter(value);
            },
            color: textColorSecondary,
            rotate: 0,
            margin: 16, // Add margin between labels
            padding: [4, 8, 4, 8], // Add padding around labels
            interval: (index: number) => {
              // Always show first and last labels
              if (index === 0 || index === data.length - 1) {
                return true;
              }
              // For intermediate labels, show every nth label
              const step = Math.max(1, Math.floor(data.length / 3));
              return index % step === 0;
            },
            hideOverlap: true, // Let ECharts handle overlap
          },
          axisPointer: {
            type: 'line',
            show: layout === 'horizontal' || (valueType === 'number' || valueType === 'duration' || valueType === 'percent'),
            triggerOn: 'mousemove',
          },
          axisLine: {
            show: false,
          },
          axisTick: {
            show: false,
          },
          splitLine: layout === "vertical" ? {
            show: true,
            lineStyle: {
              color: borderColor,
            },
          } : undefined,
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
          type: layout === "horizontal" ? "value" as const : (isTimestampData ? "time" as const : "category" as const),
          data: layout === "vertical" && !isTimestampData ? xAxisData : undefined,
          show: true,
          axisLabel: {
            show: true, // Always show labels
            formatter: (value: any) => {
              if (valueType === "percent" && layout === "horizontal") {
                return `${(value * 100).toFixed(0)}%`;
              }
              return valueFormatter(value);
            },
            color: textColorSecondary,
            margin: 16, // Add margin between labels
            padding: [4, 8, 4, 8], // Add padding around labels
            interval: (index: number) => {
              // Always show first and last labels
              if (index === 0 || index === data.length - 1) {
                return true;
              }
              // For intermediate labels, show every nth label
              const step = Math.max(1, Math.floor(data.length / 3));
              return index % step === 0;
            },
            hideOverlap: true, // Let ECharts handle overlap
          },
          axisPointer: {
            type: 'line',
            show: layout === 'vertical' || (valueType === 'number' || valueType === 'duration' || valueType === 'percent'),
            triggerOn: 'mousemove',
          },
          axisLine: {
            show: false,
          },
          axisTick: {
            show: false,
          },
          splitLine: layout === "horizontal" ? {
            show: true,
            lineStyle: {
              color: borderColor,
            },
          } : undefined,
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
      layout,
      type,
      min,
      max,
      categoryColors,
      xAxisLabel,
      yAxisLabel,
      extraDataByIndexAxis,
    ]);

    // Event handlers for the EChart component
    const chartEvents = React.useMemo(() => {
      const isTimestampData = indexType === "date" || indexType === "timestamp" || indexType === "hour" || indexType === "month" || indexType === "year" || indexType === "time";

      return {
        // Handle hover state
        mouseover: (params: any) => {
          // Handle hover on series for both horizontal and vertical charts
          if (params.componentType === 'series') {
            let indexValue: any;

            if (isTimestampData) {
              // For time axis, the timestamp is in params.value[0]
              indexValue = Array.isArray(params.value) ? params.value[0] : params.value;
            } else {
              // For category axis, use the same logic as tooltip formatter
              // For horizontal charts: use dataIndex to get the actual index value
              // For vertical charts: use axisValue directly since categories are on y-axis
              if (layout === "horizontal") {
                const dataIndex = params.dataIndex;
                indexValue = dataIndex >= 0 && dataIndex < data.length ? data[dataIndex][index] : params.name;
              } else {
                // For vertical layout, use axisValue which contains the category name
                indexValue = params.axisValue;
              }
            }

            setHoverState(indexValue, chartId, indexType);
          }
        },
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
          if (hoveredChartId === chartId) {
            setHoverState(null, null, null);
          }
        },
      };
    }, [layout, indexType, data, index, chartId, setHoverState, hoveredChartId]);

    // Handle chart instance reference
    const handleChartReady = useCallback((chart: echarts.ECharts) => {
      chartRef.current = chart;
    }, []);

    // Handle reference line updates
    useEffect(() => {
      if (!chartRef.current) return;

      const chart = chartRef.current;
      const isTimestampData = indexType === "date" || indexType === "timestamp" || indexType === "hour" || indexType === "month" || indexType === "year" || indexType === "time";

      // Handle reference line
      if (hoveredIndex != null && hoveredIndexType === indexType && hoveredChartId !== chartId) {
        const markLine = {
          silent: true,
          symbol: 'none',
          label: {
            show: false, // Hide any labels on the reference line
          },
          lineStyle: {
            color: getThemeColors().referenceLineColor,
          },
          data: [
            layout === "horizontal"
              ? { xAxis: hoveredIndex }
              : { yAxis: hoveredIndex },
          ],
        };

        // Update only the series with markLine, not the entire chart
        chart.setOption({
          series: categories.map((category) => {
            if (isTimestampData) {
              return {
                name: category,
                type: 'bar',
                barGap: '3%',
                stack: type === "stacked" ? "stack" : undefined,
                data: data.map((item) => [item[index], item[category]]),
                markLine,
              };
            } else {
              return {
                name: category,
                type: 'bar',
                barGap: '3%',
                stack: type === "stacked" ? "stack" : undefined,
                data: data.map((item) => item[category]),
                markLine,
              };
            }
          })
        });
      } else {
        // Remove markLine when not hovering
        chart.setOption({
          series: categories.map((category) => {
            if (isTimestampData) {
              return {
                name: category,
                type: 'bar',
                barGap: '3%',
                stack: type === "stacked" ? "stack" : undefined,
                data: data.map((item) => [item[index], item[category]]),
                markLine: undefined,
              };
            } else {
              return {
                name: category,
                type: 'bar',
                barGap: '3%',
                stack: type === "stacked" ? "stack" : undefined,
                data: data.map((item) => item[category]),
                markLine: undefined,
              };
            }
          })
        });
      }
    }, [hoveredIndex, hoveredChartId, hoveredIndexType, chartId, indexType, layout, categories, data, index, type]);

    return (
      <div
        ref={forwardedRef}
        className={cx("h-80 w-full relative group", className)}
        {...other}
      >
        {/* Chart container */}
        <EChart
          className="absolute inset-0"
          option={chartOptions}
          events={chartEvents}
          onChartReady={handleChartReady}
        />
        {/* Download button */}
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

BarChart.displayName = "BarChart";

export { BarChart, type BarChartProps };