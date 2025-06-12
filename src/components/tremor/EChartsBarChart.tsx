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

// Function to detect if dark mode is active
const isDarkMode = (): boolean => {
  // Check if the body has dark mode classes or if we're in a dark theme context
  const body = document.body;
  return body.classList.contains('dark') || 
         body.classList.contains('dark-mode') ||
         window.matchMedia('(prefers-color-scheme: dark)').matches;
};

// Function to get theme-appropriate colors
const getThemeColors = () => {
  const isDark = isDarkMode();
  
  if (isDark) {
    return {
      backgroundColor: getComputedCssValue('--shaper-dark-mode-background-color'),
      borderColor: getComputedCssValue('--shaper-dark-mode-border-color'),
      textColor: getComputedCssValue('--shaper-dark-mode-text-color'),
      textColorSecondary: getComputedCssValue('--shaper-dark-mode-text-color-secondary'),
      referenceLineColor: getComputedCssValue('--shaper-reference-line-color'),
    };
  } else {
    return {
      backgroundColor: getComputedCssValue('--shaper-background-color'),
      borderColor: getComputedCssValue('--shaper-border-color'),
      textColor: getComputedCssValue('--shaper-text-color'),
      textColorSecondary: getComputedCssValue('--shaper-text-color-secondary'),
      referenceLineColor: getComputedCssValue('--shaper-reference-line-color'),
    };
  }
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
    const [isChartHovered, setIsChartHovered] = React.useState(false);
    const [currentTheme, setCurrentTheme] = React.useState<'light' | 'dark'>(isDarkMode() ? 'dark' : 'light');

    const { hoveredIndex, hoveredChartId, hoveredIndexType, setHoverState } =
      React.useContext(ChartHoverContext);

    const categoryColors = constructCategoryColors(categories, AvailableChartColors);

    // Memoize the chart options to prevent unnecessary re-renders
    const chartOptions = React.useMemo(() => {
      // Get computed colors for theme
      const { backgroundColor, borderColor, textColor, textColorSecondary } = getThemeColors();

      // Check if we're dealing with timestamps
      const isTimestampData = indexType === "date" || indexType === "timestamp" || indexType === "hour" || indexType === "month" || indexType === "year" || indexType === "time";
      
      // Set up chart options
      const series: echarts.BarSeriesOption[] = categories.map((category) => {
        if (isTimestampData) {
          // For time axis, we need to provide data as [timestamp, value] pairs for both layouts
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

      return {
        animation: false,
        // Quality settings for sharper rendering
        progressive: 0, // Disable progressive rendering for better quality
        progressiveThreshold: 0,
        // Anti-aliasing and rendering quality
        renderer: 'canvas',
        useDirtyRect: true,
        tooltip: {
          show: showTooltip,
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
        axisPointer: {
          type: 'line',
          triggerOn: 'mousemove',
        },
        legend: {
          show: showLegend,
          type: enableLegendSlider ? 'scroll' : 'plain',
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
          type: layout === "horizontal" ? (isTimestampData ? "time" : "category") : "value",
          data: layout === "horizontal" && !isTimestampData ? xAxisData : undefined,
          show: showXAxis,
          axisPointer: {
            type: 'line',
            triggerOn: 'mousemove',
          },
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
          nameGap: 45,
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
          axisPointer: {
            type: 'line',
            triggerOn: 'mousemove',
          },
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
          nameGap: 60,
          nameTextStyle: {
            color: textColor,
          },
          min: minValue,
          max: maxValue,
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
      xAxisLabel,
      yAxisLabel,
      extraDataByIndexAxis,
      enableLegendSlider,
      currentTheme,
    ]);

    // Listen for theme changes
    useEffect(() => {
      const checkTheme = () => {
        const newTheme = isDarkMode() ? 'dark' : 'light';
        if (newTheme !== currentTheme) {
          setCurrentTheme(newTheme);
        }
      };

      // Check theme on mount
      checkTheme();

      // Listen for theme changes
      const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
      mediaQuery.addEventListener('change', checkTheme);

      // Also listen for class changes on body (for manual theme toggles)
      const observer = new MutationObserver(checkTheme);
      observer.observe(document.body, { attributes: true, attributeFilter: ['class'] });

      return () => {
        mediaQuery.removeEventListener('change', checkTheme);
        observer.disconnect();
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
        devicePixelRatio: window.devicePixelRatio || 1,
        renderer: 'canvas',
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
        // Only handle hover on series for horizontal charts to prevent flickering on vertical charts
        if (params.componentType === 'series' && layout === "horizontal") {
          let indexValue: any;
          
          if (isTimestampData && layout === "horizontal") {
            // For time axis, the timestamp is in params.value[0]
            indexValue = Array.isArray(params.value) ? params.value[0] : params.value;
          } else if (layout === "horizontal") {
            // For category axis, use the data index to get the actual index value
            const dataIndex = params.dataIndex;
            indexValue = dataIndex >= 0 && dataIndex < data.length ? data[dataIndex][index] : params.name;
          } else {
            // For vertical layout
            indexValue = Array.isArray(params.value) ? params.value[0] : params.value;
          }
          
          setHoverState(indexValue, chartId, indexType);
        }
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
    }, [chartInstance.current, layout, indexType, data, index, chartId, setHoverState, hoveredChartId]);

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
                stack: type === "stacked" || type === "percent" ? "stack" : undefined,
                data: data.map((item) => [item[index], item[category]]),
                markLine,
              };
            } else {
              return {
                name: category,
                type: 'bar',
                stack: type === "stacked" || type === "percent" ? "stack" : undefined,
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
                stack: type === "stacked" || type === "percent" ? "stack" : undefined,
                data: data.map((item) => [item[index], item[category]]),
                markLine: undefined,
              };
            } else {
              return {
                name: category,
                type: 'bar',
                stack: type === "stacked" || type === "percent" ? "stack" : undefined,
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
        className={cx("h-80 w-full relative", className)} 
        onMouseEnter={() => setIsChartHovered(true)}
        onMouseLeave={() => setIsChartHovered(false)}
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
              isChartHovered ? "opacity-100" : "opacity-0",
              "focus:outline-none focus:ring-2 focus:ring-cprimary dark:focus:ring-dprimary",
              "pointer-events-auto"
            )}
            onClick={() => {
              if (chartInstance.current) {
                const url = chartInstance.current.getDataURL({
                  type: 'png',
                  pixelRatio: 2,
                  backgroundColor: getThemeColors().backgroundColor
                });
                const link = document.createElement('a');
                
                // Generate a simple filename
                const timestamp = new Date().toISOString().slice(0, 10); // YYYY-MM-DD
                const filename = label 
                  ? `${label.replace(/[^a-z0-9]/gi, '_').toLowerCase()}-${timestamp}.png`
                  : `chart-${chartId}-${timestamp}.png`;
                
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