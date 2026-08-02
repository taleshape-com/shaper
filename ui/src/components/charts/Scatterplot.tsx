// SPDX-License-Identifier: MPL-2.0

import React, { useEffect, useCallback, useRef } from "react";
import type { ECharts } from "echarts/core";
import type { ScatterSeriesOption } from "echarts/charts";
import {
  constructCategoryColors,
  getThemeColors,
  getChartFont,
  getDisplayFont,
} from "../../lib/chartUtils";
import { cx } from "../../lib/utils";
import { ChartHoverContext } from "../../contexts/ChartHoverContext";
import { DarkModeContext } from "../../contexts/DarkModeContext";
import { Column, isDatableType, MarkLine } from "../../lib/types";
import { formatValue, echartsEncode } from "../../lib/render";
import { safeColor } from "../../lib/safeColor";
import { EChart } from "./EChart";

interface ScatterplotProps extends React.HTMLAttributes<HTMLDivElement> {
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
  markLines?: MarkLine[];
}

const chartPadding = 16;

const Scatterplot = (props: ScatterplotProps) => {
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
  const isHovering = React.useMemo(() => {
    if (hoveredIndex != null && hoveredIndexType === indexType && hoveredChartId != null && hoveredChartId !== chartId) {
      return hoveredIndex;
    }
    return null;
  }, [hoveredIndex, hoveredIndexType, indexType, hoveredChartId, chartId]);

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

    const isTimestampData = isDatableType(indexType);
    const isNumericData = indexType === "number" || indexType === "percent";
    const isContinuousData = isTimestampData || isNumericData;

    // Set up chart options
    const series: ScatterSeriesOption[] = [];
    categories.forEach((category) => {
      const col = categoryColors.get(category) || "#333";
      series.push({
        name: category,
        id: category,
        type: "scatter" as const,
        data: isContinuousData
          ? data.map((item) => [item[index], item[category]])
          : data.map((item) => item[category]),
        symbol: "circle",
        symbolSize: data.length > 1 ? 10 : 12,
        cursor: "crosshair",
        zlevel: 10,
        emphasis: {
          scale: 1.3,
          focus: "self",
        },
        animationDelay: 100,
        animationDelayUpdate: 100,
        itemStyle: {
          color: col,
          borderWidth: 0,
        },
      });
    });

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
        type: "scatter" as const,
        markLine: {
          silent: true,
          animation: false,
          symbol: "none",
          data: markLines.map(m => {
            return {
              xAxis: m.isYAxis ? undefined : m.value,
              yAxis: m.isYAxis ? m.value : undefined,
              symbol: m.isYAxis ? "none" : "circle",
              symbolSize: 6.5,
              lineStyle: {
                color: textColorSecondary,
                type: "dashed",
                width: m.isYAxis ? 1.2 : 1.0,
                opacity: m.isYAxis ? 0.5 : 0.8,
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
      } as any);
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
    const legendTopOffset = (showLegend ? (legendWidth / numLegendItems >= minLegendItemWidth ? 40 : 58) : 0);
    const labelTopOffset = label ? 40 + 15 * (Math.ceil(label.length / (0.125 * chartWidth)) - 1) : 10;
    const spaceForXaxisLabel = 10 + (xAxisLabel ? 25 : 0);
    const xData = !isContinuousData ? data.map((item) => item[index]) : undefined;
    const xSpace = (chartWidth - 2 * chartPadding + (yAxisLabel ? 50 : 30));
    const shortenLabel = xData ? (xSpace / xData.length) * (0.10 + (0.00004 * xSpace)) : true;

    return {
      title: {
        text: label,
        textStyle: {
          fontSize: 15,
          lineHeight: 15,
          fontFamily: displayFont,
          fontWeight: 600,
          color: textColor,
          width: chartWidth - 10 - 2 * chartPadding,
          overflow: "break",
        },
        left: "center",
        top: chartPadding,
      },
      tooltip: {
        show: true,
        trigger: "axis",
        triggerOn: "mousemove",
        enterable: false,
        confine: true,
        hideDelay: 200,
        showDelay: 0,
        borderRadius: 5,
        borderWidth: 0,
        backgroundColor: backgroundColorSecondary,
        textStyle: {
          fontFamily: chartFont,
          color: textColor,
        },
        formatter: (params: any) => {
          let indexValue: any;

          const axisData = params.find((item: any) => item?.axisDim === "x");
          if (isContinuousData) {
            indexValue = axisData.value[0]; // timestamp is the first element
          } else {
            indexValue = axisData.axisValue;
          }
          const extraData = extraDataByIndexAxis[indexValue];

          const getExtraDataForCategory = (categoryName: string) => {
            if (!extraData) return null;
            const firstVal = Object.values(extraData)[0];
            if (firstVal && typeof firstVal === "object" && !Array.isArray(firstVal)) {
              return extraData[categoryName];
            }
            if (categoryName === "" || categories.length <= 1) {
              return extraData;
            }
            return null;
          };

          const globalFields = new Set<string>();
          const fieldValuesAcrossCategories: Record<string, string[]> = {};
          const fieldRawData: Record<string, [any, Column["type"]]> = {};

          if (extraData) {
            const firstVal = Object.values(extraData)[0];
            const isNested = firstVal && typeof firstVal === "object" && !Array.isArray(firstVal);

            if (isNested) {
              Object.values(extraData).forEach((fieldsObj) => {
                if (fieldsObj && typeof fieldsObj === "object") {
                  Object.entries(fieldsObj).forEach(([fieldName, valueData]) => {
                    if (Array.isArray(valueData) && valueData.length >= 2) {
                      const [value, columnType] = valueData;
                      const formatted = formatValue(value, columnType, true).toString();
                      if (!fieldValuesAcrossCategories[fieldName]) {
                        fieldValuesAcrossCategories[fieldName] = [];
                        fieldRawData[fieldName] = valueData as [any, Column["type"]];
                      }
                      fieldValuesAcrossCategories[fieldName].push(formatted);
                    }
                  });
                }
              });

              const totalCategoriesWithExtra = Object.keys(extraData).filter(c => c !== "").length;
              Object.entries(fieldValuesAcrossCategories).forEach(([fieldName, list]) => {
                const allSame = list.every(val => val === list[0]);
                const isGlobal = allSame && (totalCategoriesWithExtra <= 1 || list.length === totalCategoriesWithExtra);
                if (isGlobal) {
                  globalFields.add(fieldName);
                }
              });
            }
          }

          const formattedIndex = indexFormatter(indexType === "duration" ? new Date(indexValue).getTime() : indexValue, xSpace / 6.5);
          let tooltipContent = `<div class="text-sm font-medium">${echartsEncode(formattedIndex)}</div>`;

          let hasGlobalExtra = false;
          let globalExtraContent = "";
          if (extraData) {
            const firstVal = Object.values(extraData)[0];
            const isNested = firstVal && typeof firstVal === "object" && !Array.isArray(firstVal);

            if (isNested) {
              globalFields.forEach((fieldName) => {
                const valueData = fieldRawData[fieldName];
                if (valueData) {
                  const [value, columnType] = valueData;
                  if (value) {
                    hasGlobalExtra = true;
                    globalExtraContent += `<div class="flex justify-between space-x-2">
                      <span class="font-medium">${echartsEncode(fieldName)}</span>
                      <span>${echartsEncode(formatValue(value, columnType, true))}</span>
                    </div>`;
                  }
                }
              });
            } else {
              Object.entries(extraData).forEach(([key, valueData]) => {
                if (Array.isArray(valueData) && valueData.length >= 2) {
                  const [value, columnType] = valueData;
                  if (value) {
                    hasGlobalExtra = true;
                    globalExtraContent += `<div class="flex justify-between space-x-2">
                      <span class="font-medium">${echartsEncode(key)}</span>
                      <span>${echartsEncode(formatValue(value, columnType, true))}</span>
                    </div>`;
                  }
                }
              });
            }
          }

          if (hasGlobalExtra) {
            tooltipContent += `<div class="mt-2">${globalExtraContent}</div>`;
          }

          tooltipContent += "<div class=\"mt-2\">";
          params.forEach((param: any) => {
            if (param.axisDim !== "x") {
              return;
            }

            let value: number;
            if (isContinuousData && Array.isArray(param.value) && param.value.length >= 2) {
              value = param.value[1] as number;
            } else {
              value = param.value as number;
            }

            if (value === null || value === undefined || isNaN(value)) {
              return;
            }

            const formattedValue = valueFormatter(value);
            tooltipContent += `<div class="flex items-center justify-between space-x-2">
              <div class="flex items-center space-x-2">
                <span class="inline-block size-2 rounded-sm" style="background-color: ${safeColor(param.color)}"></span>
                ${categories.length > 1 ? `<span>${echartsEncode(param.seriesName)}</span>` : ""}
              </div>
              <span class="font-medium">${echartsEncode(formattedValue)}</span>
            </div>`;

            const catExtraData = getExtraDataForCategory(param.seriesName);
            if (catExtraData) {
              Object.entries(catExtraData).forEach(([key, valueData]) => {
                if (globalFields.has(key)) {
                  return;
                }
                if (Array.isArray(valueData) && valueData.length >= 2) {
                  const [value, columnType] = valueData;
                  if (!value) {
                    return;
                  }
                  tooltipContent += `<div class="flex items-center justify-between space-x-2 pl-4 text-xs text-muted-foreground opacity-80 mt-0.5">
                    <span>${echartsEncode(key)}</span>
                    <span class="font-medium">${echartsEncode(formatValue(value, columnType, true))}</span>
                  </div>`;
                }
              });
            }
          });
          tooltipContent += "</div>";

          return tooltipContent;
        },
      },
      legend: {
        show: showLegend,
        data: categories,
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
        itemStyle: {
          opacity: 1,
          borderWidth: 0,
        },
        itemGap: legendItemGap,
        itemHeight: 8,
        itemWidth: 16,
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
        width: "auto",
        height: categories.length > 4 ? 40 : 20,
      },
      grid: {
        left: (yAxisLabel ? 45 : 15) + chartPadding,
        right: 10 + chartPadding + (yAxisLabel ? 20 : 0),
        top: 10 + legendTopOffset + labelTopOffset + chartPadding,
        bottom: (xAxisLabel ? 32 : 8) + chartPadding,
        outerBoundsMode: "same",
        outerBoundsContain: "axisLabel",
      },
      xAxis: {
        type: isTimestampData ? "time" as const : isNumericData ? "value" as const : "category" as const,
        data: xData,
        show: true,
        scale: isNumericData ? true : undefined,
        axisLabel: {
          show: true,
          formatter: (value: any) => {
            return indexFormatter(indexType === "duration" || indexType === "time" ? new Date(value).getTime() : value, shortenLabel);
          },
          color: textColorSecondary,
          fontFamily: chartFont,
          fontSize: 12,
          rotate: !xAxisLabel && typeof shortenLabel === "number" && shortenLabel <= 12 ? 45 : 0,
          padding: [4, 8, 4, 8],
          hideOverlap: true,
        },
        axisPointer: {
          type: "line",
          show: true,
          triggerOn: "mousemove",
          lineStyle: {
            color: referenceLineColor,
            type: "solid",
            width: 0.65,
          },
          label: {
            show: true,
            formatter: (params: any) => {
              return indexFormatter(indexType === "number" && params.value > 1 ? Math.round(params.value) : indexType === "duration" ? new Date(params.value).getTime() : params.value, xSpace / 5.8);
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
        splitLine: {
          show: false,
        },
        name: xAxisLabel,
        nameLocation: "middle",
        nameGap: 36,
        nameTextStyle: {
          color: textColor,
          fontFamily: chartFont,
          fontWeight: 500,
          fontSize: 12,
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
          type: data.length > 1 ? "line" : "none",
          show: data.length > 1,
          triggerOn: "mousemove",
          triggerEmphasis: false,
          label: {
            show: true,
            formatter: (params: any) => {
              return valueFormatter(valueType === "number" && params.value > 1 ? Math.round(params.value) : params.value);
            },
            fontFamily: chartFont,
            margin: 10,
            color: textColorInverted,
            backgroundColor: backgroundColorInverted,
          },
          lineStyle: {
            color: referenceLineColor,
            type: "solid",
            width: 0.65,
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
      graphic: {
        type: "text",
        rotation: Math.PI / 2,
        y: (chartHeight + labelTopOffset + legendTopOffset - spaceForXaxisLabel) / 2,
        x: 5 + chartPadding,
        cursor: "default",
        style: {
          text: yAxisLabel,
          font: `500 12px ${chartFont}`,
          fill: textColor,
          width: chartHeight,
          textAlign: "center",
        },
      },
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
    isDarkMode,
    chartWidth,
    chartHeight,
    label,
    markLines,
  ]);

  useEffect(() => {
    const chart = chartRef.current;
    if (!chart) {
      return;
    }
    const { referenceLineColor } = getThemeColors(isDarkMode);
    const series = [{
      id: "shaper-hover-reference-line",
      type: "scatter" as const,
      markLine: {
        animationDuration: 100,
        animationDurationUpdate: 100,
        silent: true,
        symbol: "none",
        animation: false,
        label: {
          show: false,
        },
        lineStyle: {
          color: referenceLineColor,
          type: "solid",
          width: 0.65,
        },
        data: isHovering != null && (data.length !== 1 || data[0][index] === isHovering) ? [{ xAxis: isHovering }] : [],
      },
    }];
    chart.setOption({ series }, { lazyUpdate: true });
  }, [
    data,
    index,
    isDarkMode,
    isHovering,
  ]);

  // Event handlers for the EChart component
  const chartEvents = React.useMemo(() => {
    return {
      showTip: (params: any) => {
        let indexValue: any;

        if (params.dataIndex !== undefined && params.seriesIndex !== undefined) {
          const dataIndex = params.dataIndex;
          if (dataIndex >= 0 && dataIndex < data.length) {
            indexValue = data[dataIndex][index];
          }
        } else if (params.axisValue !== undefined) {
          indexValue = params.axisValue;
        }

        if (indexValue !== undefined) {
          setHoverState(indexValue, chartId, indexType);
        }
      },
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

Scatterplot.displayName = "Scatterplot";

export { Scatterplot, type ScatterplotProps };
