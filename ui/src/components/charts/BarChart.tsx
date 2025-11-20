// SPDX-License-Identifier: MPL-2.0

import React, { useEffect, useCallback, useRef } from "react";
import type { ECharts } from "echarts/core";
import type { BarSeriesOption } from "echarts/charts";
import {
  constructCategoryColors,
  getThemeColors,
  getChartFont,
  getDisplayFont,
} from "../../lib/chartUtils";
import { cx } from "../../lib/utils";
import { ChartHoverContext } from "../../contexts/ChartHoverContext";
import { DarkModeContext } from "../../contexts/DarkModeContext";
import { Column, isDatableType, isTimeType, MarkLine } from "../../lib/types";
import { formatValue, echartsEncode } from "../../lib/render";
import { translate } from "../../lib/translate";
import { EChart } from "./EChart";

const chartPadding = 16;

interface BarChartProps extends React.HTMLAttributes<HTMLDivElement> {
  chartId: string;
  label?: string;
  data: Record<string, any>[];
  extraDataByIndexAxis: Record<string, Record<string, any>>;
  index: string;
  indexType: Column["type"];
  valueType: Column["type"];
  categories: string[];
  colorsByCategory: Record<string, string>;
  valueFormatter: (value: number, shortFormat?: boolean | number) => string;
  indexFormatter: (value: number | string, shortFormat?: boolean | number) => string;
  showLegend?: boolean;
  xAxisLabel?: string;
  yAxisLabel?: string;
  layout: "vertical" | "horizontal";
  type: "default" | "stacked";
  markLines?: MarkLine[];
}

const BarChart = (props: BarChartProps) => {
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
    layout,
    type,
    chartId,
    label,
    markLines,
    ...other
  } = props;

  const chartRef = useRef<ECharts | null>(null);
  const hoveredChartIdRef = useRef<string | null>(null);
  const [chartWidth, setChartWidth] = React.useState(450);
  const [chartHeight, setChartHeight] = React.useState(300);

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
    const { borderColor, textColor, textColorSecondary, referenceLineColor, backgroundColorSecondary, textColorInverted, backgroundColorInverted } = getThemeColors(isDarkMode);
    const chartFont = getChartFont();
    const displayFont = getDisplayFont();
    const categoryColors = constructCategoryColors(categories, colorsByCategory, isDarkMode);
    const isTimestampData = isDatableType(indexType) || indexType === "number";

    // We treat vertical timestamp data as categories.
    let dataCopy = data;
    if (layout === "vertical" && isDatableType(indexType)) {
      dataCopy = data.map((item) => {
        return {
          ...item,
          // Since we treat index as category, it will be converted to text. Cannot keep date in the original number format.
          [index]: new Date(item[index]).toISOString(),
        };
      });
    }

    // Set up chart options
    const series: BarSeriesOption[] = categories.map((category) => ({
      name: category,
      id: category,
      type: "bar" as const,
      barGap: "3%",
      barMaxWidth: dataCopy.length === 1 ? layout == "horizontal" ? "50%" : "25%" : undefined,
      stack: type === "stacked" ? "stack" : category,
      cursor: "crosshair",
      data: isTimestampData && layout === "horizontal"
        ? dataCopy.map((item) => [item[index], item[category]])
        : dataCopy.map((item) => item[category]),
      itemStyle: {
        color: categoryColors.get(category),
      },
    }));

    if (markLines) {
      let foundEventLine = false;
      let multiEventLines = false;
      for (const m of markLines) {
        const isEventLine = !m.isYAxis && layout === "horizontal";
        if (isEventLine) {
          if (foundEventLine) {
            multiEventLines = true;
            break;
          }
          foundEventLine = true;
        }
      }
      series.push({
        type: "bar" as const,
        stack: type === "stacked" ? "stack" : categories[0],
        markLine: {
          silent: true,
          symbol: "none",
          data: markLines.filter(m => !m.isYAxis || layout === "horizontal").map(m => {
            const isGoalLine = m.isYAxis || layout === "vertical";
            return {
              xAxis: m.isYAxis ? undefined : m.value,
              yAxis: m.isYAxis ? m.value : undefined,
              symbol: isGoalLine ? "none" : "circle",
              symbolSize: 6.5,
              lineStyle: {
                color: textColorSecondary,
                type: "dashed",
                width: isGoalLine ? 1.2 : 1.0,
                opacity: isGoalLine ? 0.5 : 0.8,
              },
              label: {
                formatter: m.label,
                position: isGoalLine ? "insideStartTop" : multiEventLines ? "insideEnd" : "end",
                distance: isGoalLine ? 1.5 : multiEventLines ? 5 : 3.8,
                color: textColorSecondary,
                fontFamily: chartFont,
                fontWeight: 500,
                fontSize: 11,
                opacity: isGoalLine ? 0.5 : 0.8,
                width: m.isYAxis ? chartWidth / 3 : chartHeight / 2,
                overflow: "truncate",
                textBorderColor: backgroundColorSecondary,
                textBorderWidth: !isGoalLine && multiEventLines ? 2 : 0,
              },
            };
          }),
        },
      });
    }

    const numLegendItems = categories.filter(c => c.length > 0).length;
    const avgLegendCharCount = categories.reduce((acc, c) => acc + c.length, 0) / numLegendItems;
    const minLegendItemWidth = Math.max(avgLegendCharCount * 8, 50);
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
    const xData = layout === "horizontal" && !isTimestampData ? dataCopy.map((item) => item[index]) : undefined;
    const xSpace = (chartWidth - 2 * chartPadding + (yAxisLabel ? 50 : 30));
    const shortenLabel = layout === "horizontal" ? xData ? (xSpace / xData.length) * (0.10 + (0.00004 * xSpace)) : true : false;
    let maxLabelLen = 0;
    (xData ?? []).forEach(x => {
      const v = layout === "horizontal" ? indexFormatter(indexType === "duration" || indexType === "time" ? new Date(x).getTime() : x, shortenLabel) : valueFormatter(x, true);
      if (v.length > maxLabelLen) {
        maxLabelLen = v.length;
      }
    });
    const shouldRotateXLabel = !xAxisLabel && typeof shortenLabel === "number" && shortenLabel <= 12;
    let customValues = undefined;
    if (layout === "horizontal" && isTimestampData) {
      const canFitAll = (chartWidth - 2 * chartPadding + (yAxisLabel ? 20 : 0)) / dataCopy.length > (isTimeType(indexType) ? 80 : indexType === "duration" ? 75 : 45);
      if (canFitAll) {
        customValues = dataCopy.map((item) => item[index]);
      } else {
        const numVals = Math.floor(xSpace / 130);
        const dataMin = Math.min(...data.map(d => d[index]));
        const dataMax = Math.max(...data.map(d => d[index]));
        const dataPadding = (dataMax - dataMin) * (60 / xSpace);
        const dataSpan = (dataMax - dataMin) - 2 * dataPadding;
        const offset = dataSpan / numVals;
        customValues = [];
        for (let i = 0; i <= numVals; i++) {
          customValues.push(Math.round(dataMin + dataPadding + i * offset));
        }
      }
    }

    return {
      animation: false,
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
        trigger: "axis",
        triggerOn: "mousemove",
        enterable: false,
        confine: true,
        hideDelay: 200, // Increase hide delay to prevent flickering
        showDelay: 0, // Show immediately
        borderRadius: 5,
        borderWidth: 0,
        backgroundColor: backgroundColorSecondary,
        textStyle: {
          fontFamily: chartFont,
          color: textColor,
        },
        formatter: (params: any) => {
          const indexDim = layout === "horizontal" ? "x" : "y";
          const axisData = params.find((item: any) => item?.axisDim === indexDim);
          const hoverValue = axisData?.axisValue;
          const title = indexFormatter(indexType === "duration" || indexType === "time" ? new Date(hoverValue).getTime() : hoverValue);

          let tooltipContent = `<div class="text-sm font-medium">${echartsEncode(title)}</div>`;

          if (type === "stacked" && (valueType === "number" || valueType === "duration")) {
            const total = params.reduce((sum: number, item: any) => {
              if (item.axisDim !== indexDim) {
                return sum; // Skip non-index axis items
              }
              let value: number;
              if (isTimestampData && layout === "horizontal" && Array.isArray(item.value) && item.value.length >= 2) {
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
              <span class="font-medium">${echartsEncode(translate("Total"))}</span>
              <span>${echartsEncode(valueFormatter(total))}</span>
            </div>`;
          }

          const extraData = extraDataByIndexAxis[hoverValue];
          if (extraData) {
            tooltipContent += "<div class=\"mt-2\">";
            Object.entries(extraData).forEach(([key, valueData]) => {
              if (Array.isArray(valueData) && valueData.length >= 2) {
                const [value, columnType] = valueData;
                tooltipContent += `<div class="flex justify-between space-x-2">
                  <span class="font-medium">${echartsEncode(key)}</span>
                  <span>${echartsEncode(formatValue(value, columnType, true))}</span>
                </div>`;
              }
            });
            tooltipContent += "</div>";
          }

          // Use a Set to track shown categories
          tooltipContent += "<div class=\"mt-2\">";
          params.forEach((param: any) => {
            if (param.axisDim !== indexDim) {
              return; // Skip non-index axis items
            }
            let value: number;
            if (isTimestampData && layout === "horizontal" && Array.isArray(param.value) && param.value.length >= 2) {
              value = param.value[1] as number;
            } else {
              value = param.value as number;
            }

            // Skip categories with missing or null values
            if (value === null || value === undefined || isNaN(value)) {
              return;
            }

            const formattedValue = valueFormatter(value);
            tooltipContent += `<div class="flex items-center justify-between space-x-2">
              <div class="flex items-center space-x-2">
                <span class="inline-block size-2 rounded-sm" style="background-color: ${echartsEncode(param.color)}"></span>
                ${categories.length > 1 ? `<span>${echartsEncode(param.seriesName)}</span>` : ""}
              </div>
              <span class="font-medium">${echartsEncode(formattedValue)}</span>
            </div>`;
          });
          tooltipContent += "</div>";

          return tooltipContent;
        },
      },
      legend: {
        show: showLegend,
        selectedMode: false,
        type: canFitLegendItems ? "plain" : "scroll",
        orient: "horizontal",
        left: chartPadding,
        top: 7 + labelTopOffset + chartPadding,
        padding: [5, canFitLegendItems ? legendPaddingRight : 25, 5, legendPaddingLeft],
        textStyle: {
          color: textColor,
          fontFamily: chartFont,
          fontWeight: 500,
          width: canFitLegendItems ? legendItemWidth : undefined,
          overflow: "truncate",
        },
        itemGap: legendItemGap,
        itemHeight: 10,
        itemWidth: 10,
        pageButtonPosition: "end",
        pageButtonGap: 10,
        pageButtonItemGap: 5,
        pageIconColor: textColorSecondary,
        pageIconInactiveColor: borderColor,
        pageIcons: {
          horizontal: [
            "M10.8284 12.0007L15.7782 16.9504L14.364 18.3646L8 12.0007L14.364 5.63672L15.7782 7.05093L10.8284 12.0007Z",
            "M13.1717 12.0007L8.22192 7.05093L9.63614 5.63672L16.0001 12.0007L9.63614 18.3646L8.22192 16.9504L13.1717 12.0007Z",
          ],
        },
        pageIconSize: 12,
        pageFormatter: () => "",
        pageTextStyle: {
          fontSize: 1,
        },
      },
      grid: {
        left: (yAxisLabel ? 45 : 15) + chartPadding,
        right: 15 + chartPadding,
        top: 10 + legendTopOffset + labelTopOffset + chartPadding,
        bottom: (xAxisLabel ? 35 : 10) + chartPadding,
        outerBoundsMode: "same",
        outerBoundsContain: "axisLabel",
      },
      xAxis: {
        type: layout === "vertical" ? "value" as const : (isTimestampData ? "time" as const : "category" as const),
        data: xData,
        show: true,
        axisLabel: {
          show: true,
          formatter: (value: any) => {
            if (layout === "horizontal") {
              return indexFormatter(indexType === "duration" || indexType === "time" ? new Date(value).getTime() : value, shortenLabel);
            }
            return valueFormatter(value, true);
          },
          color: textColorSecondary,
          fontFamily: chartFont,
          fontSize: 12,
          interval: xData && !shouldRotateXLabel ? Math.floor((maxLabelLen / 13) * xData.length / (chartWidth / 80)) : undefined,
          rotate: shouldRotateXLabel ? 45 : 0,
          hideOverlap: true,
          //customValues,
          padding: [4, 8, 4, 8],
        },
        axisPointer: {
          type: "line",
          show: layout === "horizontal" || dataCopy.length > 1,
          triggerEmphasis: layout === "horizontal",
          triggerOn: "mousemove",
          triggerTooltip: layout === "horizontal",
          lineStyle: {
            color: referenceLineColor,
            type: "dashed",
            width: 0.8,
          },
          label: {
            show: data.length > 1,
            formatter: (params: any) => {
              if (layout === "horizontal") {
                return indexFormatter(indexType === "number" && params.value > 1 ? Math.round(params.value) : indexType === "duration" || indexType === "time" ? new Date(params.value).getTime() : params.value);
              }
              return valueFormatter(valueType === "number" && params.value > 1 ? Math.round(params.value) : params.value, true);
            },
            fontFamily: chartFont,
            margin: 5,
            color: textColorInverted,
            backgroundColor: backgroundColorInverted,
          },
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
        nameLocation: "middle",
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
            return indexFormatter(indexType === "duration" || indexType === "time" ? new Date(value).getTime() : value, xSpace / 30);
          },
          color: textColorSecondary,
          fontFamily: chartFont,
          fontSize: 12,
          showMinLabel: true,
          showMaxLabel: true,
          padding: [4, 8, 4, 8], // Add padding around labels
          hideOverlap: true,
        },
        axisPointer: {
          type: "line",
          show: layout === "vertical" || dataCopy.length > 1,
          triggerEmphasis: layout === "vertical",
          triggerOn: "mousemove",
          lineStyle: {
            color: referenceLineColor,
            type: "dashed",
            width: 0.8,
          },
          label: {
            show: layout === "horizontal" || dataCopy.length > 1,
            formatter: (params: any) => {
              if (layout === "horizontal") {
                return valueFormatter(valueType === "number" && params.value > 1 ? Math.round(params.value) : params.value);
              }
              return indexFormatter(indexType === "number" && params.value > 1 ? Math.round(params.value) : indexType === "duration" || indexType === "time" ? new Date(params.value).getTime() : params.value);
            },
            fontFamily: chartFont,
            margin: 10,
            color: textColorInverted,
            backgroundColor: backgroundColorInverted,
          },
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
      // We use a graphic element to display the y-axis label instead of the axis name
      // since the name can overlap with the axis labels.
      // See https://github.com/apache/echarts/issues/12415#issuecomment-2285226567
      graphic: {
        type: "text",
        rotation: Math.PI / 2,
        y: (chartHeight + labelTopOffset + legendTopOffset - spaceForXaxisLabel) / 2,
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
    layout,
    type,
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
      setIsHovering(hoveredIndex);
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
    const isTimestampData = isDatableType(indexType) || indexType === "number";
    const series: BarSeriesOption[] = [{
      id: "shaper-hover-reference-line",
      type: "bar" as const,
      stack: type === "stacked" ? "stack" : categories[0],
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
        data: isHovering != null ? [{
          [layout === "horizontal" ? "xAxis" : "yAxis"]: isTimestampData && layout === "vertical"
            ? indexType === "number" ? isHovering.toString() : new Date(isHovering).toISOString()
            : isHovering,
        }] : [],
      },
    }];
    chart.setOption({ series }, { lazyUpdate: true });
  }, [
    categories,
    indexType,
    isDarkMode,
    layout,
    type,
    isHovering,
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

BarChart.displayName = "BarChart";

export { BarChart, type BarChartProps };
