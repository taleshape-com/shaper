import React, { useCallback, useRef } from "react";
import { Column, Result } from "../../lib/dashboard";
import { EChart } from "../charts/EChart";
import * as echarts from 'echarts';
import { getThemeColors, getChartFont } from '../../lib/chartUtils';
import { DarkModeContext } from '../../contexts/DarkModeContext';
import { formatValue } from "../../lib/render";

export type GaugeCategory = {
  from: number;
  to: number;
  label: string;
  color: string;
};

type DashboardGaugeProps = {
  headers: Column[];
  data: Result['sections'][0]['queries'][0]['rows'];
  gaugeCategories: GaugeCategory[];
  label?: string;
};

const DashboardGauge: React.FC<DashboardGaugeProps> = ({ headers, data, gaugeCategories, label }) => {
  const chartRef = useRef<echarts.ECharts | null>(null);
  const { isDarkMode } = React.useContext(DarkModeContext);

  const chartOptions = React.useMemo((): echarts.EChartsOption => {
    const theme = getThemeColors(isDarkMode);
    const chartFont = getChartFont();

    // Find the value column
    const valueIndex = headers.findIndex(h => h.tag === 'value');
    const valueHeader = headers[valueIndex];
    const value = data[0][valueIndex];

    // Calculate all unique boundary values (from and to)
    const boundaryValues = Array.from(new Set([
      ...gaugeCategories.map(cat => cat.from),
      ...gaugeCategories.map(cat => cat.to),
    ])).sort((a, b) => a - b);

    // Calculate min/max and boundaries
    const min = boundaryValues[0];
    const max = boundaryValues[boundaryValues.length - 1];

    // Color stops for axisLine
    const colorStops = gaugeCategories.map(cat => [
      (cat.to - min) / (max - min),
      cat.color
    ]) as [number, string][];

    // Helper to check if a value is a boundary (with float tolerance)
    function isBoundary(val: number) {
      return boundaryValues.some(b => Math.abs(b - val) < 1e-6);
    }

    // axisLabel formatter: only show value at boundaries
    function valueLabelFormatter(v: number) {
      return isBoundary(v) ? v.toString() : '';
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
      console.log(v, gaugeCategories[i].label);
      return gaugeCategories[i].label;
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
      radius: '90%',
    }

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
            length: '8%',
            width: 13,
            offsetCenter: [0, '-79%'],
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
  ]);

  const handleChartReady = useCallback((chart: echarts.ECharts) => {
    chartRef.current = chart;
  }, []);

  return (
    <div className="w-full h-full flex flex-col items-center justify-center">
      <EChart
        className="absolute inset-0"
        option={chartOptions}
        onChartReady={handleChartReady}
      />
    </div>
  );
};

export default DashboardGauge;