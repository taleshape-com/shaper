import React, { useEffect, useRef, useCallback } from "react";
import * as echarts from "echarts";
import { RiDownload2Line } from "@remixicon/react";
import {
  AvailableEChartsColors,
  constructEChartsCategoryColors,
  getEChartsColor,
  isDarkMode,
  getThemeColors,
  downloadChartAsImage,
} from "../../lib/chartUtils";
import { cx } from "../../lib/utils";
import { ChartHoverContext } from "../../contexts/ChartHoverContext";
import { Column } from "../../lib/dashboard";
import { formatValue } from "../../lib/render";
import { translate } from "../../lib/translate";

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

    const chartRef = useRef<HTMLDivElement>(null);
    const chartInstance = useRef<echarts.ECharts | null>(null);
    const [currentTheme, setCurrentTheme] = React.useState<'light' | 'dark'>(isDarkMode() ? 'dark' : 'light');

    const { hoveredIndex, hoveredChartId, hoveredIndexType, setHoverState } =
      React.useContext(ChartHoverContext);

    const categoryColors = constructEChartsCategoryColors(categories, AvailableEChartsColors);

    // Memoized download handler
    const handleDownload = useCallback(() => {
      if (chartInstance.current) {
        downloadChartAsImage(chartInstance.current, chartId, label);
      }
    }, [chartId, label, chartInstance]);

    // Memoize the chart options to prevent unnecessary re-renders
    const chartOptions = React.useMemo(() => {
      // Get computed colors for theme
      const { backgroundColor, borderColor, textColor, textColorSecondary } = getThemeColors();
      const isDark = isDarkMode();

      // Check if we're dealing with timestamps
      const isTimestampData = indexType === "date" || indexType === "timestamp" || indexType === "hour" || indexType === "month" || indexType === "year" || indexType === "time";

      // Set up chart options
      const series: echarts.LineSeriesOption[] = categories.map((category) => {
        if (isTimestampData) {
          // For time axis, we need to provide data as [timestamp, value] pairs
          return {
            name: category,
            type: 'line',
            data: data.map((item) => [item[index], item[category]]),
            connectNulls: true,
            symbol: 'circle',
            symbolSize: 6,
            showSymbol: false,
            lineStyle: {
              color: getEChartsColor(categoryColors.get(category) || 'primary', isDark),
              width: 2,
            },
            itemStyle: {
              color: getEChartsColor(categoryColors.get(category) || 'primary', isDark),
              borderColor: backgroundColor,
              borderWidth: 2,
            },
            emphasis: {
              showSymbol: true,
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
          };
        } else {
          // For category axis, use the original format
          return {
            name: category,
            type: 'line',
            data: data.map((item) => item[category]),
            connectNulls: true,
            symbol: 'circle',
            symbolSize: 6,
            showSymbol: false,
            lineStyle: {
              color: getEChartsColor(categoryColors.get(category) || 'primary', isDark),
              width: 2,
            },
            itemStyle: {
              color: getEChartsColor(categoryColors.get(category) || 'primary', isDark),
              borderColor: backgroundColor,
              borderWidth: 2,
            },
            emphasis: {
              showSymbol: true,
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
          type: isTimestampData ? "time" : "category",
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
          type: "value",
          show: true,
          axisLabel: {
            show: true,
            formatter: (value: any) => {
              return valueFormatter(value);
            },
            color: textColorSecondary,
            margin: 16,
            padding: [4, 8, 4, 8],
            interval: (index: number) => {
              // Always show first and last labels
              if (index === 0 || index === 5) {
                return true;
              }
              // For intermediate labels, show every nth label
              const step = Math.max(1, Math.floor(5 / 3));
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
      valueFormatter,
      indexFormatter,
      showLegend,
      xAxisLabel,
      yAxisLabel,
      min,
      max,
      categoryColors,
      extraDataByIndexAxis,
    ]);

    // Listen for theme changes
    useEffect(() => {
      const checkTheme = () => {
        const newTheme = isDarkMode() ? 'dark' : 'light';
        if (newTheme !== currentTheme) {
          setCurrentTheme(newTheme);
        }
      };
      checkTheme();
      const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
      mediaQuery.addEventListener('change', checkTheme);
      return () => {
        mediaQuery.removeEventListener('change', checkTheme);
      };
    }, [currentTheme]);

    // Create a callback ref that handles both refs
    const setRefs = React.useCallback((node: HTMLDivElement | null) => {
      // Set the forwarded ref
      if (typeof forwardedRef === "function") {
        forwardedRef(node);
      } else if (forwardedRef) {
        (forwardedRef as React.MutableRefObject<HTMLDivElement | null>).current = node;
      }
      // Update our ref
      (chartRef as React.MutableRefObject<HTMLDivElement | null>).current = node;
    }, [forwardedRef]);

    useEffect(() => {
      if (!chartRef.current) return;

      // Initialize chart with device pixel ratio for sharp rendering
      const chart = echarts.init(chartRef.current, undefined, {
        useDirtyRect: true, // Optimize rendering performance
      });
      chartInstance.current = chart;

      // Set chart options
      chart.setOption(chartOptions);

      // Handle resize
      const handleResize = () => {
        chart.resize();
      };
      window.addEventListener('resize', handleResize);

      return () => {
        window.removeEventListener('resize', handleResize);
        chart.dispose();
      };
    }, [
      chartOptions,
      chartRef,
      chartInstance,
    ]);

    // Separate useEffect for event handlers
    useEffect(() => {
      if (!chartInstance.current) return;

      const chart = chartInstance.current;
      const isTimestampData = indexType === "date" || indexType === "timestamp" || indexType === "hour" || indexType === "month" || indexType === "year" || indexType === "time";

      // Handle hover state
      chart.on('mouseover', (params: any) => {
        let indexValue: any;

        if (isTimestampData) {
          // For time axis, the timestamp is in params.value[0]
          indexValue = Array.isArray(params.value) ? params.value[0] : params.value;
        } else {
          // For category axis, use the data index to get the actual index value
          const dataIndex = params.dataIndex;
          indexValue = dataIndex >= 0 && dataIndex < data.length ? data[dataIndex][index] : params.name;
        }

        setHoverState(indexValue, chartId, indexType);
      });

      // Add tooltip event handler
      chart.on('showTip', (params: any) => {
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
      });

      // Also handle tooltip hide to clear hover state
      chart.on('hideTip', () => {
        if (hoveredChartId === chartId) {
          setHoverState(null, null, null);
        }
      });

      return () => {
        // Clean up event handlers
        chart.off('mouseover');
        chart.off('showTip');
        chart.off('hideTip');
      };
    }, [chartInstance.current, indexType, data, index, chartId, setHoverState, hoveredChartId]);

    // Separate useEffect for hover state updates to prevent chart recreation
    useEffect(() => {
      if (!chartInstance.current) return;

      const chart = chartInstance.current;
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
            { xAxis: hoveredIndex },
          ],
        };

        // Update only the series with markLine, not the entire chart
        chart.setOption({
          series: categories.map((category) => {
            if (isTimestampData) {
              return {
                name: category,
                type: 'line',
                data: data.map((item) => [item[index], item[category]]),
                connectNulls: true,
                markLine,
              };
            } else {
              return {
                name: category,
                type: 'line',
                data: data.map((item) => item[category]),
                connectNulls: true,
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
                type: 'line',
                data: data.map((item) => [item[index], item[category]]),
                connectNulls: true,
                markLine: undefined,
              };
            } else {
              return {
                name: category,
                type: 'line',
                data: data.map((item) => item[category]),
                connectNulls: true,
                markLine: undefined,
              };
            }
          })
        });
      }
    }, [hoveredIndex, hoveredChartId, hoveredIndexType, chartId, indexType, categories, data, index]);

    return (
      <div
        className={cx("h-80 w-full relative group", className)}
        {...other}
      >
        {/* Chart container */}
        <div
          ref={setRefs}
          className="absolute inset-0"
          style={{
            imageRendering: 'crisp-edges',
          }}
        />
        {/* Button container */}
        <div className="absolute inset-0 pointer-events-none">
          <button
            className={cx(
              "absolute right-2 z-10",
              showLegend ? "top-7" : "top-2",
              "p-1.5 rounded-md",
              "bg-cbg dark:bg-dbg",
              "border border-cb dark:border-db",
              "text-ctext dark:text-dtext",
              "hover:bg-cbgs dark:hover:bg-dbgs",
              "transition-all duration-100",
              "opacity-0 group-hover:opacity-100",
              "focus:outline-none focus:ring-2 focus:ring-cprimary dark:focus:ring-dprimary",
              "pointer-events-auto"
            )}
            onClick={handleDownload}
            title={translate('Save as image')}
          >
            <RiDownload2Line className="size-4" />
          </button>
        </div>
      </div>
    );
  }
);

LineChart.displayName = "LineChart";

export { LineChart, type LineChartProps };