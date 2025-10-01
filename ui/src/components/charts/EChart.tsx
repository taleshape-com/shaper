// SPDX-License-Identifier: MPL-2.0

import { useMemo, useRef, useEffect } from "react";
import { debounce } from "lodash";

import * as echarts from "echarts/core";
import { BarChart, LineChart, GaugeChart } from "echarts/charts";
import {
  TitleComponent,
  TooltipComponent,
  DatasetComponent,
  TransformComponent,
  AxisPointerComponent,
  GraphicComponent,
  GridComponent,
  GridSimpleComponent,
  LegendComponent,
  LegendPlainComponent,
  LegendScrollComponent,
  MarkLineComponent,
} from "echarts/components";
import { LabelLayout, UniversalTransition } from "echarts/features";
import { CanvasRenderer, SVGRenderer } from "echarts/renderers";

interface EChartProps {
  option: echarts.EChartsCoreOption;
  events?: Record<string, (param: any) => void>;
  onChartReady?: (chart: echarts.ECharts) => void;
  onResize?: (chart: echarts.ECharts) => void;
  [key: string]: any;
}

echarts.use([
  BarChart,
  LineChart,
  GaugeChart,
  TitleComponent,
  TooltipComponent,
  GridComponent,
  DatasetComponent,
  TransformComponent,
  AxisPointerComponent,
  GraphicComponent,
  GridComponent,
  GridSimpleComponent,
  LegendComponent,
  LegendPlainComponent,
  LegendScrollComponent,
  MarkLineComponent,
  LabelLayout,
  UniversalTransition,
  // SVG renderer as default it looks sharper
  // and allows zooming in browser and PDFs
  SVGRenderer,
  // Using canvas renderer to support downloading as PNG
  CanvasRenderer,
]);

const optionSettings = {
  replaceMerge: "series",
  lazyUpdate: true,
};

export const EChart = ({
  option,
  events = {},
  onChartReady,
  onResize,
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
            if (onResize) {
              onResize(chart);
            }
          }
        }
      }, 50),
    [onResize],
  );

  useEffect(() => {
    if (!chartRef.current) return;
    const chart = echarts.init(chartRef.current, null, { renderer: "svg" });
    if (onChartReady) {
      onChartReady(chart);
    }
    const resizeObserver = new ResizeObserver(() => {
      resizeChart();
    });
    resizeObserver.observe(chartRef.current);

    // TODO: I am not sure if this is needed and if it even does anything
    const handlePrint = () => {
      if (chartRef.current) {
        const chart = echarts.getInstanceByDom(chartRef.current);
        if (chart) {
          chart.resize();
          if (onResize) {
            onResize(chart);
          }
        }
      }
    };
    window.addEventListener("beforeprint", handlePrint);
    window.addEventListener("afterprint", handlePrint);

    const currentRef = chartRef.current;
    return () => {
      chart?.dispose();
      if (currentRef) {
        resizeObserver.unobserve(currentRef);
      }
      resizeObserver.disconnect();
      window.removeEventListener("beforeprint", handlePrint);
      window.removeEventListener("afterprint", handlePrint);
    };
  }, [resizeChart, onChartReady, onResize]);

  useEffect(() => {
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
      if (typeof handler === "function") {
        chart.on(key, handler);
      }
    });

    // Update previous event keys
    prevEventKeysRef.current = currentEventKeys;
  }, [events]);

  useEffect(() => {
    if (!chartRef.current) return;
    const chart = echarts.getInstanceByDom(chartRef.current);
    if (chart) {
      chart.setOption(option, optionSettings);
    }
  }, [option]);

  return (
    <div
      ref={chartRef}
      style={{ imageRendering: "crisp-edges" }}
      {...props}
    />
  );
};
