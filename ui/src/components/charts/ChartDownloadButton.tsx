// SPDX-License-Identifier: MPL-2.0

import React from 'react';
import * as echarts from 'echarts/core';
import type { ECharts } from 'echarts/core';
import { RiDownload2Line } from "@remixicon/react";
import { downloadChartAsImage } from "../../lib/chartUtils";
import { cx } from "../../lib/utils";
import { translate } from "../../lib/translate";
import { DarkModeContext } from "../../contexts/DarkModeContext";

interface ChartDownloadButtonProps {
  chartId: string;
  label?: string;
  className?: string;
}

export const ChartDownloadButton: React.FC<ChartDownloadButtonProps> = ({
  chartId,
  label,
  className,
}) => {
  const { isDarkMode } = React.useContext(DarkModeContext);
  const handleDownload = React.useCallback(() => {
    let chart: ECharts | null = null;
    const chartElement = document.querySelector(`[data-chart-id="${chartId}"]`) as HTMLElement;
    if (chartElement) {
      const instances = echarts.getInstanceByDom(chartElement);
      if (instances) {
        chart = instances;
      }
    }
    if (chart) {
      downloadChartAsImage(chart, isDarkMode, chartId, label);
      return;
    }
    console.warn(`Could not find chart element with id: ${chartId}`);
  }, [chartId, label, isDarkMode]);

  return (
    <div className="absolute inset-0 pointer-events-none print:hidden">
      <button
        className={cx(
          "absolute top-2 right-2 z-50",
          "p-1.5 rounded-md",
          "bg-cbg dark:bg-dbg",
          "border border-cb dark:border-db",
          "text-ctext dark:text-dtext",
          "hover:bg-cbgs dark:hover:bg-dbgs",
          "transition-all duration-100",
          "opacity-0 group-hover:opacity-100",
          "focus:outline-none focus:ring-2 focus:ring-cprimary dark:focus:ring-dprimary",
          "pointer-events-auto",
          className
        )}
        onClick={handleDownload}
        title={translate('Save as image')}
      >
        <RiDownload2Line className="size-4" />
      </button>
    </div>
  );
};