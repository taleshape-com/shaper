import React, { useEffect, useCallback, useRef } from "react";
import * as echarts from "echarts";
import {
  AvailableEChartsColors,
  constructEChartsCategoryColors,
  getEChartsColor,
  getThemeColors,
  getChartFont,
} from "../../lib/chartUtils";
import { cx } from "../../lib/utils";
import { ChartHoverContext } from "../../contexts/ChartHoverContext";
import { DarkModeContext } from "../../contexts/DarkModeContext";
import { Column, isTimeType } from "../../lib/dashboard";
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
  valueFormatter: (value: number, shortFormat?: boolean) => string;
  indexFormatter: (value: number, shortFormat?: boolean) => string;
  showLegend?: boolean;
  xAxisLabel?: string;
  yAxisLabel?: string;
  layout: "vertical" | "horizontal";
  type: "default" | "stacked";
  label?: string;
}

const BarChart = (props: BarChartProps) => {
  const {
    data,
    extraDataByIndexAxis,
    categories,
    index,
    indexType,
    valueType,
    valueFormatter,
    indexFormatter,
    showLegend = true,
    className,
    xAxisLabel,
    yAxisLabel,
    layout,
    type,
    chartId,
    label,
    ...other
  } = props;

  const chartRef = useRef<echarts.ECharts | null>(null);
  const hoveredChartIdRef = useRef<string | null>(null);

  const { hoveredIndex, hoveredChartId, hoveredIndexType, setHoverState } =
    React.useContext(ChartHoverContext);

  const { isDarkMode } = React.useContext(DarkModeContext);

  // Update hoveredChartId ref whenever it changes
  useEffect(() => {
    hoveredChartIdRef.current = hoveredChartId;
  }, [hoveredChartId]);

  const categoryColors = constructEChartsCategoryColors(categories, AvailableEChartsColors);

  // Memoize the chart options to prevent unnecessary re-renders
  const chartOptions = React.useMemo((): echarts.EChartsOption => {
    // Get computed colors for theme
    const { borderColor, textColor, textColorSecondary, referenceLineColor } = getThemeColors(isDarkMode);
    const isDark = isDarkMode;
    const chartFont = getChartFont();

    // Check if we're dealing with timestamps
    // TODO: I am still not completely sure why we need to handle time as timestamp as well
    const isTimestampData = isTimeType(indexType) || indexType === "time";

    // We treat vertical timestamp data as categories.
    let dataCopy = data;
    if (isTimestampData && (layout === 'vertical' || data.length < 2)) {
      dataCopy = data.map((item) => {
        return {
          ...item,
          [index]: indexFormatter(item[index])
        };
      });
    }

    // Set up chart options
    const series: echarts.BarSeriesOption[] = categories.map((category) => {
      const baseSeries: echarts.BarSeriesOption = {
        name: category,
        id: category,
        type: 'bar' as const,
        barGap: '3%',
        barMaxWidth: dataCopy.length === 1 ? layout == 'horizontal' ? '50%' : '25%' : undefined,
        stack: type === "stacked" ? "stack" : undefined,
        data: isTimestampData && layout === "horizontal" && data.length > 1
          ? dataCopy.map((item) => [item[index], item[category]])
          : dataCopy.map((item) => item[category]),
        itemStyle: {
          color: getEChartsColor(categoryColors.get(category) || 'primary', isDark),
        },
        emphasis: {
          itemStyle: {
            color: getEChartsColor(categoryColors.get(category) || 'primary', isDark),
            opacity: dataCopy.length > 1 ? 0.8 : 1,
          },
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
              layout === "horizontal"
                ? { xAxis: hoveredIndex }
                : { yAxis: hoveredIndex },
            ],
          },
        };
      }

      return baseSeries;
    });

    return {
      animation: false,
      // Quality settings for sharper rendering
      progressive: 0, // Disable progressive rendering for better quality
      progressiveThreshold: 0,
      // Anti-aliasing and rendering quality
      // renderer: 'canvas',
      useDirtyRect: true,
      cursor: 'default',
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
          const indexDim = layout === 'horizontal' ? 'x' : 'y';
          const axisData = params.find((item: any) => item?.axisDim === indexDim);
          const hoverValue = axisData?.axisValue;

          const title = layout === 'horizontal'
            ? isTimestampData && data.length < 2
              ? hoverValue
              : indexFormatter(hoverValue)
            : isTimestampData
              ? hoverValue
              : indexFormatter(hoverValue);

          let tooltipContent = `<div class="text-sm font-medium">${title}</div>`;

          if (type === "stacked" && (valueType === "number" || valueType === "duration")) {
            const total = params.reduce((sum: number, item: any) => {
              if (item.axisDim !== indexDim) {
                return sum; // Skip non-index axis items
              }
              let value: number;
              if (isTimestampData && layout === 'horizontal' && Array.isArray(item.value) && item.value.length >= 2) {
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

          const extraData = extraDataByIndexAxis[hoverValue];
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
            if (param.axisDim !== indexDim) {
              return; // Skip non-index axis items
            }
            let value: number;
            if (isTimestampData && layout === 'horizontal' && Array.isArray(param.value) && param.value.length >= 2) {
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
        type: 'scroll',
        orient: 'horizontal',
        left: 0,
        top: 7,
        padding: [5, 25, 5, 5],
        textStyle: {
          color: textColor,
          fontFamily: chartFont,
          fontWeight: 500,
        },
        itemGap: 12,
        itemHeight: 10,
        itemWidth: 10,
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
      },
      grid: {
        left: yAxisLabel ? 45 : 15,
        right: 15,
        top: showLegend ? 50 : 20,
        bottom: xAxisLabel ? 35 : 10,
        containLabel: true,
      },
      xAxis: {
        type: layout === "horizontal" ? (isTimestampData && data.length > 1 ? "time" as const : "category" as const) : "value" as const,
        data: layout === "horizontal" && (!isTimestampData || data.length < 2) ? dataCopy.map((item) => item[index]) : undefined,
        show: true,
        axisLabel: {
          show: true, // Always show labels
          formatter: (value: any) => {
            if (layout === "horizontal") {
              if (isTimestampData && data.length < 2) {
                return value;
              }
              return indexFormatter(value, true);
            }
            return valueFormatter(value, true);
          },
          color: textColorSecondary,
          fontFamily: chartFont,
          padding: [4, 8, 4, 8], // Add padding around labels
          hideOverlap: true,
        },
        axisPointer: {
          type: layout === 'vertical' || dataCopy.length > 1 ? 'line' : 'none',
          show: layout === 'horizontal' || dataCopy.length > 1,
          triggerOn: 'mousemove',
          triggerTooltip: layout === "horizontal",
          label: {
            show: data.length > 1,
            formatter: (params: any) => {
              if (layout === "horizontal") {
                return indexFormatter(indexType === "number" && params.value > 1 ? Math.round(params.value) : params.value);
              }
              return valueFormatter(valueType === "number" && params.value > 1 ? Math.round(params.value) : params.value);
            },
            fontFamily: chartFont,
          }
        },
        axisLine: {
          show: false,
        },
        axisTick: {
          show: false,
        },
        splitLine: layout === "vertical" ? {
          show: dataCopy.length > 1,
          lineStyle: {
            color: borderColor,
          },
        } : undefined,
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
        type: layout === "horizontal" ? "value" as const : ("category" as const),
        data: layout === "vertical" ? dataCopy.map((item) => item[index]) : undefined,
        show: true,
        axisLabel: {
          show: true, // Always show labels
          formatter: (value: any) => {
            if (layout === "horizontal") {
              return valueFormatter(value, true);
            }
            if (isTimestampData) {
              return value
            }
            return indexFormatter(value, true);
          },
          color: textColorSecondary,
          fontFamily: chartFont,
          padding: [4, 8, 4, 8], // Add padding around labels
          hideOverlap: true,
        },
        axisPointer: {
          type: layout === 'horizontal' || dataCopy.length > 1 ? 'line' : 'none',
          show: layout === 'vertical' || dataCopy.length > 1,
          triggerOn: 'mousemove',
          label: {
            show: layout === 'horizontal' || dataCopy.length > 1,
            formatter: (params: any) => {
              if (layout === "horizontal") {
                return valueFormatter(valueType === "number" && params.value > 1 ? Math.round(params.value) : params.value);
              }
              if (isTimestampData) {
                return params.value;
              }
              return indexFormatter(indexType === "number" && params.value > 1 ? Math.round(params.value) : params.value);
            },
            fontFamily: chartFont,
          }
        },
        axisLine: {
          show: false,
        },
        axisTick: {
          show: false,
        },
        splitLine: layout === "horizontal" ? {
          show: dataCopy.length > 1,
          lineStyle: {
            color: borderColor,
          },
        } : undefined,
      },
      series,
      graphic: [
        // We use a graphic element to display the y-axis label instead of the axis name
        // since the name can overlap with the axis labels.
        // See https://github.com/apache/echarts/issues/12415#issuecomment-2285226567
        {
          type: "text",
          left: 5,
          top: "center",
          rotation: Math.PI / 2,
          style: {
            text: yAxisLabel,
            font: `500 14px ${chartFont}`,
            fill: textColor,
          }
        },
      ],
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
    categoryColors,
    xAxisLabel,
    yAxisLabel,
    extraDataByIndexAxis,
    hoveredIndex,
    hoveredIndexType,
    hoveredChartId,
    chartId,
    isDarkMode,
  ]);

  // Event handlers for the EChart component
  const chartEvents = React.useMemo(() => {
    return {
      // Add tooltip event handler
      showTip: (params: any) => {
        let dataIndex = -1;
        let indexValue: any;

        if (layout === "vertical") {
          // Safely access nested properties for vertical layout
          const dataByCoordSys = Array.isArray(params?.dataByCoordSys) && params.dataByCoordSys.length > 0
            ? params.dataByCoordSys[0]
            : undefined;
          if (dataByCoordSys?.dataByAxis) {
            const yAxisData = dataByCoordSys.dataByAxis.find((item: any) => item?.axisDim === "y");
            if (yAxisData?.value !== undefined) {
              dataIndex = yAxisData.value;
            }
          }
        } else {
          // For horizontal layout, use standard params
          if (params?.dataIndex !== undefined && params?.seriesIndex !== undefined) {
            dataIndex = params.dataIndex;
          } else if (params?.axisValue !== undefined) {
            indexValue = params.axisValue;
          }
        }

        // Only try to get indexValue from data if we have a valid dataIndex
        if (dataIndex >= 0 && dataIndex < data.length) {
          const item = data[dataIndex];
          if (item && index in item) {
            indexValue = item[index];
          }
        }

        if (indexValue !== undefined) {
          setHoverState(indexValue, chartId, indexType);
        }
      },
      // Also handle tooltip hide to clear hover state
      hideTip: () => {
        // Always clear hover state when tooltip is hidden
        if (hoveredChartIdRef.current === chartId) {
          setHoverState(null, null, null);
        }
      },
    };
  }, [data, index, chartId, setHoverState, indexType, layout]);

  // Handle chart instance reference
  const handleChartReady = useCallback((chart: echarts.ECharts) => {
    chartRef.current = chart;
  }, []);

  return (
    <div
      className={cx("h-full w-full relative group select-none", className)}
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
      />
    </div>
  );
};

BarChart.displayName = "BarChart";

export { BarChart, type BarChartProps };