import React from 'react';
import * as echarts from 'echarts';
import { RiDownload2Line } from "@remixicon/react";
import { downloadChartAsImage } from "../../lib/chartUtils";
import { cx } from "../../lib/utils";
import { translate } from "../../lib/translate";

interface ChartDownloadButtonProps {
  chartRef: React.RefObject<echarts.ECharts | null>;
  chartId: string;
  label?: string;
  showLegend?: boolean;
  className?: string;
}

export const ChartDownloadButton: React.FC<ChartDownloadButtonProps> = ({
  chartRef,
  chartId,
  label,
  showLegend = true,
  className,
}) => {
  const handleDownload = React.useCallback(() => {
    if (chartRef.current) {
      downloadChartAsImage(chartRef.current, chartId, label);
    }
  }, [chartId, label, chartRef]);

  return (
    <div className="absolute inset-0 pointer-events-none">
      <button
        className={cx(
          "absolute -right-2 z-10",
          label ? "-top-11" : "-top-2",
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