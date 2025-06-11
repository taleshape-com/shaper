import React, { useEffect, useRef } from "react";
import * as echarts from "echarts";
import { RiDownload2Line } from "@remixicon/react";
import {
  AvailableChartColors,
  constructCategoryColors,
} from "../../lib/chartUtils";
import { cx } from "../../lib/utils";
import { ChartHoverContext } from "../../contexts/ChartHoverContext";
import { Column } from "../../lib/dashboard";
import { formatValue } from "../../lib/render";
import { translate } from "../../lib/translate";

// Map of color keys to CSS variable names
const colorKeyToCssVar: Record<string, string> = {
  primary: '--shaper-primary-color',
  color2: '--shaper-color-two',
  color3: '--shaper-color-three',
  color4: '--shaper-color-four',
  color5: '--shaper-color-five',
  color6: '--shaper-color-six',
  color7: '--shaper-color-seven',
  color8: '--shaper-color-eight',
  color9: '--shaper-color-nine',
  color10: '--shaper-color-ten',
  violet: '--shaper-color-nine', // Using color-nine for violet
  blue: '--shaper-color-eight', // Using color-eight for blue
  fuchsia: '--shaper-color-three', // Using color-three for fuchsia
  amber: '--shaper-color-six', // Using color-six for amber
  cyan: '--shaper-color-two', // Using color-two for cyan
  gray: '--shaper-color-ten', // Using color-ten for gray
  lime: '--shaper-color-seven', // Using color-seven for lime
};

// Function to get the actual color value from a color key
const getColorValue = (colorKey: string): string => {
  const cssVar = colorKeyToCssVar[colorKey];
  if (!cssVar) return '#000000'; // Fallback to black if color key not found
  
  // Get the computed value of the CSS variable from the root element
  const root = document.documentElement;
  const computedValue = getComputedStyle(root).getPropertyValue(cssVar).trim();
  
  // If we can't get the computed value, fall back to using the CSS variable directly
  return computedValue || `var(${cssVar})`;
};

// Function to get computed CSS value
const getComputedCssValue = (cssVar: string): string => {
  const root = document.documentElement;
  const computedValue = getComputedStyle(root).getPropertyValue(cssVar).trim();
  return computedValue || `var(${cssVar})`;
};

interface EChartsBarChartProps extends React.HTMLAttributes<HTMLDivElement> {
  chartId: string;
  data: Record<string, any>[];
  extraDataByIndexAxis: Record<string, Record<string, any>>;
  index: string;
  indexType: Column['type'];
  valueType: Column['type'];
  categories: string[];
  valueFormatter?: (value: number) => string;
  indexFormatter?: (value: number) => string;
  showXAxis?: boolean;
  showYAxis?: boolean;
  showGridLines?: boolean;
  showTooltip?: boolean;
  showLegend?: boolean;
  minValue?: number;
  maxValue?: number;
  enableLegendSlider?: boolean;
  xAxisLabel?: string;
  yAxisLabel?: string;
  layout?: "vertical" | "horizontal";
  type?: "default" | "stacked" | "percent";
  indexAxisDomain?: [string | number, string | number];
  label?: string;
}

const EChartsBarChart = React.forwardRef<HTMLDivElement, EChartsBarChartProps>(
  (props, forwardedRef) => {
    const {
      data = [],
      extraDataByIndexAxis,
      categories = [],
      index,
      indexType,
      valueType,
      valueFormatter = (value: number) => value.toString(),
      indexFormatter = (value: number) => value.toString(),
      showXAxis = true,
      showYAxis = true,
      showGridLines = true,
      showTooltip = true,
      showLegend = true,
      minValue,
      maxValue,
      className,
      enableLegendSlider = false,
      xAxisLabel,
      yAxisLabel,
      layout = "horizontal",
      type = "default",
      indexAxisDomain = ["auto", "auto"],
      chartId,
      label,
      ...other
    } = props;

    const chartRef = useRef<HTMLDivElement>(null);
    const chartInstance = useRef<echarts.ECharts | null>(null);

    const { hoveredIndex, hoveredChartId, hoveredIndexType, setHoverState } =
      React.useContext(ChartHoverContext);

    const categoryColors = constructCategoryColors(categories, AvailableChartColors);

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

      // Initialize chart
      const chart = echarts.init(chartRef.current);
      chartInstance.current = chart;

      // Get computed colors for theme
      const backgroundColor = getComputedCssValue('--shaper-background-color');
      const borderColor = getComputedCssValue('--shaper-border-color');
      const textColor = getComputedCssValue('--shaper-text-color');
      const textColorSecondary = getComputedCssValue('--shaper-text-color-secondary');

      // Debug: Log the data structure
      console.log('ECharts data:', data);
      console.log('Categories:', categories);
      console.log('Index:', index);
      console.log('Index type:', indexType);

      // Check if we're dealing with timestamps
      const isTimestampData = indexType === "date" || indexType === "timestamp" || indexType === "hour" || indexType === "month" || indexType === "year";
      
      // Set up chart options
      const series: echarts.BarSeriesOption[] = categories.map((category) => {
        if (isTimestampData && layout === "horizontal") {
          // For time axis, we need to provide data as [timestamp, value] pairs
          return {
            name: category,
            type: 'bar',
            stack: type === "stacked" || type === "percent" ? "stack" : undefined,
            data: data.map((item) => [item[index], item[category]]),
            itemStyle: {
              color: getColorValue(categoryColors.get(category) || 'primary'),
            },
            emphasis: {
              itemStyle: {
                color: getColorValue(categoryColors.get(category) || 'primary'),
                opacity: 0.8,
              },
            },
          };
        } else {
          // For category axis, use the original format
          return {
            name: category,
            type: 'bar',
            stack: type === "stacked" || type === "percent" ? "stack" : undefined,
            data: data.map((item) => item[category]),
            itemStyle: {
              color: getColorValue(categoryColors.get(category) || 'primary'),
            },
            emphasis: {
              itemStyle: {
                color: getColorValue(categoryColors.get(category) || 'primary'),
                opacity: 0.8,
              },
            },
          };
        }
      });

      // For category axis, we need to use the index values as category names
      const xAxisData = data.map((item) => item[index]);
      
      // Debug: Log the axis data
      console.log('X-axis data:', xAxisData);

      const option: echarts.EChartsOption = {
        animation: false,
        tooltip: {
          show: showTooltip,
          trigger: 'axis',
          formatter: (params: any) => {
            let indexValue: any;
            let extraData: any;
            
            if (isTimestampData && layout === "horizontal") {
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
            
            if (type === "stacked" && (valueType === "number" || valueType === "duration")) {
              const total = params.reduce((sum: number, item: any) => {
                let value: number;
                if (isTimestampData && layout === "horizontal" && Array.isArray(item.value) && item.value.length >= 2) {
                  value = item.value[1] as number;
                } else {
                  value = item.value as number;
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
              if (isTimestampData && layout === "horizontal" && Array.isArray(param.value) && param.value.length >= 2) {
                value = param.value[1] as number;
              } else {
                value = param.value as number;
              }
              const formattedValue = type === "percent" ? value * 100 + "%" : formatValue(value, valueType, true);
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
          type: enableLegendSlider ? 'scroll' : 'plain',
          orient: 'horizontal',
          top: 0,
          textStyle: {
            color: textColor,
          },
        },
        grid: {
          left: yAxisLabel ? 60 : 40,
          right: 20,
          top: showLegend ? 40 : 20,
          bottom: xAxisLabel ? 40 : 20,
          containLabel: true,
        },
        xAxis: {
          type: layout === "horizontal" ? (isTimestampData ? "time" : "category") : "value",
          data: layout === "horizontal" && !isTimestampData ? xAxisData : undefined,
          show: showXAxis,
          axisLabel: {
            show: true, // Always show labels
            formatter: (value: any) => {
              if (type === "percent") {
                return `${(value * 100).toFixed(0)}%`;
              }
              // For timestamps, format the value properly
              return indexFormatter(value);
            },
            color: textColorSecondary,
            rotate: 0,
            // For time axes, use interval to prevent overlap
            ...(isTimestampData && {
              interval: Math.ceil(data.length / 10), // Show roughly 10 labels max
              hideOverlap: true,
            }),
            // For other cases, use auto interval
            ...(!isTimestampData && {
              interval: 'auto',
            }),
          },
          axisLine: {
            show: false,
          },
          axisTick: {
            show: false,
          },
          splitLine: showGridLines && layout === "vertical" ? {
            show: true,
            lineStyle: {
              color: borderColor,
            },
          } : undefined,
          name: xAxisLabel,
          nameLocation: 'middle',
          nameGap: 25,
          nameTextStyle: {
            color: textColor,
          },
          min: indexAxisDomain[0] === "auto" ? undefined : indexAxisDomain[0],
          max: indexAxisDomain[1] === "auto" ? undefined : indexAxisDomain[1],
        },
        yAxis: {
          type: layout === "horizontal" ? "value" : (isTimestampData ? "time" : "category"),
          data: layout === "vertical" && !isTimestampData ? xAxisData : undefined,
          show: showYAxis,
          axisLabel: {
            show: true, // Always show labels
            formatter: (value: any) => {
              if (type === "percent") {
                return `${(value * 100).toFixed(0)}%`;
              }
              return valueFormatter(value);
            },
            color: textColorSecondary,
            // For time axes, use interval to prevent overlap
            ...(isTimestampData && {
              interval: Math.ceil(data.length / 10), // Show roughly 10 labels max
              hideOverlap: true,
            }),
            // For other cases, use auto interval
            ...(!isTimestampData && {
              interval: 'auto',
            }),
          },
          axisLine: {
            show: false,
          },
          axisTick: {
            show: false,
          },
          splitLine: showGridLines && layout === "horizontal" ? {
            show: true,
            lineStyle: {
              color: borderColor,
            },
          } : undefined,
          name: yAxisLabel,
          nameLocation: 'middle',
          nameGap: 40,
          nameTextStyle: {
            color: textColor,
          },
          min: minValue,
          max: maxValue,
        },
        series,
      };

      // Handle hover state
      chart.on('mouseover', (params: any) => {
        if (params.componentType === 'series') {
          const indexValue = layout === "horizontal" 
            ? params.name 
            : Array.isArray(params.value) ? params.value[0] : params.value;
          setHoverState(indexValue, chartId, indexType);
        }
      });

      chart.on('mouseout', () => {
        if (hoveredChartId === chartId) {
          setHoverState(null, null, null);
        }
      });

      // Handle reference line
      if (hoveredIndex != null && hoveredIndexType === indexType && hoveredChartId !== chartId) {
        const markLine = {
          silent: true,
          symbol: 'none',
          lineStyle: {
            color: getComputedCssValue('--shaper-reference-line-color'),
          },
          data: [
            layout === "horizontal"
              ? { xAxis: hoveredIndex }
              : { yAxis: hoveredIndex },
          ],
        };
        option.series = series.map(s => ({
          ...s,
          markLine,
        }));
      }

      // Set chart options
      chart.setOption(option);

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
      data,
      categories,
      index,
      indexType,
      valueType,
      valueFormatter,
      indexFormatter,
      showXAxis,
      showYAxis,
      showGridLines,
      showTooltip,
      showLegend,
      minValue,
      maxValue,
      layout,
      type,
      indexAxisDomain,
      categoryColors,
      chartId,
      hoveredIndex,
      hoveredChartId,
      hoveredIndexType,
      setHoverState,
      xAxisLabel,
      yAxisLabel,
      extraDataByIndexAxis,
    ]);

    return (
      <div className={cx("h-80 w-full relative", className)} {...other}>
        {/* Chart container */}
        <div
          ref={setRefs}
          className="absolute inset-0"
        />
        {/* Button container */}
        <div className="absolute inset-0 pointer-events-none">
          <button
            className={cx(
              "absolute right-2 top-2 z-10",
              "p-1.5 rounded-md",
              "bg-cbg dark:bg-dbg",
              "border border-cb dark:border-db",
              "text-ctext dark:text-dtext",
              "hover:bg-cbgs dark:hover:bg-dbgs",
              "transition-all duration-100",
              "opacity-50 hover:opacity-100",
              "focus:outline-none focus:ring-2 focus:ring-cprimary dark:focus:ring-dprimary",
              "pointer-events-auto"
            )}
            onClick={() => {
              if (chartInstance.current) {
                const url = chartInstance.current.getDataURL({
                  type: 'png',
                  pixelRatio: 2,
                  backgroundColor: getComputedCssValue('--shaper-background-color')
                });
                const link = document.createElement('a');
                // Use label if available, otherwise fall back to chartId or 'chart'
                const filename = label 
                  ? `${label.replace(/[^a-z0-9]/gi, '_').toLowerCase()}.png`
                  : `${chartId || 'chart'}.png`;
                link.download = filename;
                link.href = url;
                document.body.appendChild(link);
                link.click();
                document.body.removeChild(link);
              }
            }}
            title={translate('Save as image')}
          >
            <RiDownload2Line className="size-4" />
          </button>
        </div>
      </div>
    );
  }
);

EChartsBarChart.displayName = "EChartsBarChart";

export { EChartsBarChart, type EChartsBarChartProps }; 