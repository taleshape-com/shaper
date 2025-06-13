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
    if (!chartRef.current) return;
    const chart = echarts.init(chartRef.current, null, chartSettings);
    if (onChartReady) {
      onChartReady(chart);
    }
    for (const [key, handler] of Object.entries(events)) {
      chart.on(key, (param) => {
        if (typeof handler === 'function') {
          handler(param);
        }
      });
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
  }, [chartSettings, events, resizeChart, onChartReady]);

  useEffect(() => {
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
