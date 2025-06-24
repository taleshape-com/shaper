import React, { useCallback, useRef } from "react";
import { Column, GaugeCategory, Result } from "../../lib/dashboard";
import { EChart } from "../charts/EChart";
import * as echarts from 'echarts';
import { getThemeColors, getChartFont, AvailableEChartsColors, getEChartsColor } from '../../lib/chartUtils';
import { DarkModeContext } from '../../contexts/DarkModeContext';
import { formatValue } from "../../lib/render";

type DashboardGaugeProps = {
  headers: Column[];
  data: Result['sections'][0]['queries'][0]['rows'];
  gaugeCategories: GaugeCategory[];
  label?: string;
};

const DashboardGauge: React.FC<DashboardGaugeProps> = ({ headers, data, gaugeCategories, label }) => {
  const chartRef = useRef<echarts.ECharts | null>(null);
  const { isDarkMode } = React.useContext(DarkModeContext);
  const [chartHeight, setChartHeight] = React.useState<number>(0);


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
    })
    function categoryLabelFormatter(v: number) {
      const i = centerValues.findIndex(b => Math.abs(b - v) < 1e-6);
      if (i === -1) {
        return '';
      }
      return gaugeCategoriesWithColor[i].label ?? '';
    }

    const centerGCD = diffs.reduce((acc, val) => gcd(acc, val / 2));
    const centerNumber = Math.min((max - min) / centerGCD, 1000);

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
      center: ['50%', '75%'],
      radius: '100%',
    };

    const pointerOffset = chartHeight * -0.5 + 36;

    return {
      series: [
        {
          ...baseSeries,
          splitNumber: centerNumber,
          axisLine: {
            lineStyle: {
              width: 36,
              color: colorStops,
            },
          },
          axisLabel: {
            show: true,
            distance: -32,
            rotate: 'tangential',
            color: theme.textColorSecondary,
            fontSize: 12,
            fontWeight: 600,
            fontFamily: chartFont,
            formatter: categoryLabelFormatter,
          },
          pointer: {
            icon: 'triangle',
            length: 16,
            width: 14,
            offsetCenter: [0, pointerOffset],
            itemStyle: {
              color: theme.textColorSecondary,
              shadowBlur: 2,
              shadowColor: 'rgba(0,0,0,0.10)',
            }
          },
          detail: {
            valueAnimation: false,
            fontSize: 36,
            fontFamily: chartFont,
            offsetCenter: [0, '-7%'],
            color: theme.textColor,
            fontWeight: 600,
            formatter: function(v: number) {
              return formatValue(v, valueHeader.type, true).toString();
            },
          },
        },
        {
          ...baseSeries,
          splitNumber: splitNumber,
          axisLine: {
            show: false,
          },
          axisLabel: {
            distance: 26,
            color: theme.textColorSecondary,
            fontSize: 12,
            fontFamily: chartFont,
            formatter: valueLabelFormatter,
          },
          pointer: {
            show: false,
          },
          detail: {
            show: false,
          },
        },
      ],
    };
  }, [
    data,
    isDarkMode,
    gaugeCategories,
    headers,
    label,
    chartHeight,
  ]);

  const handleChartReady = useCallback((chart: echarts.ECharts) => {
    chartRef.current = chart;
    setChartHeight(chart.getHeight());
  }, []);

  const handleChartResize = useCallback((chart: echarts.ECharts) => {
    setChartHeight(chart.getHeight());
  }, []);

  return (
    <div className="w-full h-full relative select-none">
      <EChart
        className="absolute inset-0"
        option={chartOptions}
        onChartReady={handleChartReady}
        onResize={handleChartResize}
      />
    </div>
  );
};

export default DashboardGauge;