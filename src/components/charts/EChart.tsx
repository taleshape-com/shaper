import { useMemo, useRef, useEffect } from 'react';
import * as echarts from 'echarts';
import { debounce } from 'lodash';

interface EChartProps {
  option: echarts.EChartsOption;
  chartSettings?: echarts.EChartsInitOpts;
  optionSettings?: echarts.SetOptionOpts;
  events?: Record<string, (param: any) => void>;
  onChartReady?: (chart: echarts.ECharts) => void;
  [key: string]: any;
}

export const EChart = ({
  option,
  chartSettings,
  optionSettings,
  events = {},
  onChartReady,
  ...props
}: EChartProps) => {
  const chartRef = useRef<HTMLDivElement>(null);
  const prevEventKeysRef = useRef<string[]>([]);
  const resizeChart = useMemo(
    () =>
      debounce(() => {
        if (chartRef.current) {
          const chart = echarts.getInstanceByDom(chartRef.current);
          if (chart) {
            chart.resize();
          }
        }
      }, 50),
    []
  );

  useEffect(() => {
    console.log('init effect')
    if (!chartRef.current) return;
    const chart = echarts.init(chartRef.current, null, chartSettings);
    if (onChartReady) {
      onChartReady(chart);
    }
    const resizeObserver = new ResizeObserver(() => {
      resizeChart();
    });
    resizeObserver.observe(chartRef.current);
    const currentRef = chartRef.current;
    return () => {
      chart?.dispose();
      if (currentRef) {
        resizeObserver.unobserve(currentRef);
      }
      resizeObserver.disconnect();
    };
  }, [chartSettings, resizeChart, onChartReady]);

  useEffect(() => {
    console.log('events effect')

    if (!chartRef.current) return;
    const chart = echarts.getInstanceByDom(chartRef.current);
    if (!chart) return;

    // Remove previous event listeners
    prevEventKeysRef.current.forEach(key => {
      chart.off(key);
    });

    // Attach new event listeners
    const currentEventKeys = Object.keys(events);
    currentEventKeys.forEach(key => {
      const handler = events[key];
      if (typeof handler === 'function') {
        chart.on(key, handler);
      }
    });

    // Update previous event keys
    prevEventKeysRef.current = currentEventKeys;
  }, [events]);

  useEffect(() => {
    console.log('option effect')
    if (!chartRef.current) return;
    const chart = echarts.getInstanceByDom(chartRef.current);
    if (chart) {
      chart.setOption(option, optionSettings);
    }
  }, [option, optionSettings]);

  return (
    <div 
      ref={chartRef}
      style={{ imageRendering: 'crisp-edges' }}
      {...props}
      />
  );
};
