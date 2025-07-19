import React, { useCallback, useRef } from "react";
import { Column, GaugeCategory, Result } from "../../lib/dashboard";
import { EChart } from "../charts/EChart";
import * as echarts from 'echarts';
import { getThemeColors, getChartFont, AvailableEChartsColors, getEChartsColor } from '../../lib/chartUtils';
import { DarkModeContext } from '../../contexts/DarkModeContext';
import { formatValue } from "../../lib/render";
import { ChartDownloadButton } from "../charts/ChartDownloadButton";

type DashboardGaugeProps = {
  chartId: string;
  headers: Column[];
  data: Result['sections'][0]['queries'][0]['rows'];
  gaugeCategories: GaugeCategory[];
  label?: string;
};

const barWidth = 42;

const DashboardGauge: React.FC<DashboardGaugeProps> = ({
  chartId,
  headers,
  data,
  gaugeCategories,
  label,
}) => {
  const chartRef = useRef<echarts.ECharts | null>(null);
  const { isDarkMode } = React.useContext(DarkModeContext);
  const [chartSize, setChartSize] = React.useState<{ width: number, height: number }>({ width: 0, height: 0 });


  const chartOptions = React.useMemo((): echarts.EChartsOption => {
    const theme = getThemeColors(isDarkMode);
    const chartFont = getChartFont();

    // Find the value column
    const valueIndex = headers.findIndex(h => h.tag === 'value');
    const valueHeader = headers[valueIndex];
    const value = data[0][valueIndex];

    const gaugeCategoriesWithColor = gaugeCategories.map((cat, i) => {
      if (cat.color) return cat;
      let color = theme.borderColor;
      if (i > 0) {
        const colorKey = AvailableEChartsColors[i + 1 % AvailableEChartsColors.length];
        color = getEChartsColor(colorKey, isDarkMode);
      }
      return {
        ...cat,
        color,
      };
    });

    // Calculate all unique boundary values (from and to)
    const boundaryValues = Array.from(new Set([
      ...gaugeCategoriesWithColor.map(cat => cat.from),
      ...gaugeCategoriesWithColor.map(cat => cat.to),
    ])).sort((a, b) => a - b);

    // Calculate min/max and boundaries
    const min = boundaryValues[0];
    const max = boundaryValues[boundaryValues.length - 1];

    // Color stops for axisLine
    const colorStops = gaugeCategoriesWithColor.map(cat => [
      (cat.to - min) / (max - min),
      cat.color!
    ]) as [number, string][];

    // Helper to check if a value is a boundary (with float tolerance)
    function isBoundary(val: number) {
      return boundaryValues.some(b => Math.abs(b - val) < 1e-6);
    }

    // axisLabel formatter: only show value at boundaries
    function valueLabelFormatter(v: number) {
      return isBoundary(v) ? formatValue(v, valueHeader.type, true).toString() : '';
    }

    // Helper to calculate GCD
    function gcd(a: number, b: number): number {
      return b === 0 ? a : gcd(b, a % b);
    }

    // Calculate GCD of all differences between consecutive boundaries
    const diffs = [];
    for (let i = 1; i < boundaryValues.length; i++) {
      diffs.push(boundaryValues[i] - boundaryValues[i - 1]);
    }
    const boundaryGCD = diffs.reduce((acc, val) => gcd(acc, val));
    const splitNumber = Math.min((max - min) / boundaryGCD, 1000);

    const centerValues = diffs.map((d, i) => {
      return boundaryValues[i] + d / 2;
    });

    const baseSeries = {
      type: 'gauge' as const,
      min,
      max,
      axisTick: {
        show: false,
      },
      splitLine: {
        show: false,
      },
      title: {
        show: false,
      },
      data: [
        {
          value: typeof value === 'number' ? value : Number(value),
          name: label || valueHeader?.name || '',
        }
      ],
      startAngle: 180,
      endAngle: 0,
      center: ['50%', chartSize.width > chartSize.height ? '75%' : '68%'],
      radius: chartSize.width > 340 ? '110%' : '86%',
    };

    const gaugeRadius = chartSize.width > 340 ? 1.1 : 0.86;
    const centerY = chartSize.width > chartSize.height ? 0.75 : 0.68;
    const centerPx = [0.5 * chartSize.width, centerY * Math.min(chartSize.width, chartSize.height)];
    const r = (Math.min(chartSize.width, chartSize.height) / 2) * gaugeRadius + 9;

    // Using custom graphics to draw labels size with axisLabel we cannot control the individual alignment to ensure they don't overlap with the bar
    const graphics = (chartSize.width > 0 && chartSize.height > 0)
      ? centerValues.map((v, i) => {
        const relative = (v - min) / (max - min);
        const angle = Math.PI - (relative) * Math.PI; // 180° to 0°
        const x = centerPx[0] + r * Math.cos(angle);
        const y = centerPx[1] - r * Math.sin(angle);
        return {
          type: 'text',
          x,
          y,
          style: {
            text: gaugeCategoriesWithColor[i].label ?? '',
            fill: theme.textColorSecondary,
            font: `600 12px ${chartFont}`,
            textAlign: relative < 0.4 ? 'right' : relative > 0.6 ? 'left' : 'center',
            textVerticalAlign: 'middle',
          },
          z: 100,
          cursor: 'default',
        };
      })
      : [];

    const pointerOffset = Math.min(chartSize.width, chartSize.height) * (chartSize.width > 340 ? 1.1 : 0.86) * -0.5 + barWidth;

    return {
      animation: false,
      series: [
        {
          ...baseSeries,
          splitNumber,
          // Avoids cursor:pointer on pointer hover. Need to change if we ever want to make the gauge interactive
          silent: true,
          axisLine: {
            lineStyle: {
              width: barWidth,
              color: colorStops,
            },
          },
          axisLabel: {
            distance: 26,
            color: theme.textColorSecondary,
            fontSize: 12,
            fontFamily: chartFont,
            formatter: valueLabelFormatter,
          },
          progress: {
            show: gaugeCategories.length < 2,
            width: barWidth,
            itemStyle: {
              color: theme.primaryColor,
            }
          },
          pointer: {
            show: gaugeCategories.length >= 2,
            icon: 'triangle',
            length: 16,
            width: 14,
            offsetCenter: [0, pointerOffset],
            itemStyle: {
              color: theme.textColor,
            }
          },
          detail: {
            valueAnimation: false,
            fontSize: chartSize.width > 300 ? 36 : 24,
            fontFamily: chartFont,
            offsetCenter: [0, '-26%'],
            color: theme.textColor,
            fontWeight: 600,
            formatter: function(v: number) {
              return formatValue(v, valueHeader.type, true).toString();
            },
          },
        },
      ],
      graphic: graphics,
    };
  }, [
    data,
    isDarkMode,
    gaugeCategories,
    headers,
    label,
    chartSize,
  ]);

  const handleChartReady = useCallback((chart: echarts.ECharts) => {
    chartRef.current = chart;
    setChartSize({ width: chart.getWidth(), height: chart.getHeight() });
  }, []);

  const handleChartResize = useCallback((chart: echarts.ECharts) => {
    setChartSize({ width: chart.getWidth(), height: chart.getHeight() });
  }, []);

  return (
    <div className="w-full h-full relative group select-none">
      <EChart
        className="absolute inset-0"
        option={chartOptions}
        onChartReady={handleChartReady}
        onResize={handleChartResize}
      />
      <ChartDownloadButton
        chartRef={chartRef}
        chartId={chartId}
        label={label}
      />
    </div>
  );
};

export default DashboardGauge;