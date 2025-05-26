import { RiDownloadLine } from "@remixicon/react";
import html2canvas from "html2canvas";
import { cx } from "../../lib/utils";

interface ChartDownloadButtonProps {
  chartElement: HTMLDivElement | null;
  chartId: string;
  isVisible: boolean;
  onDownloadStart: () => void;
  onDownloadEnd: () => void;
  label?: string;
}

export const ChartDownloadButton = ({
  chartElement,
  chartId,
  isVisible,
  onDownloadStart,
  onDownloadEnd,
  label
}: ChartDownloadButtonProps) => {
  const handleDownload = async () => {
    if (!chartElement) return;

    try {
      onDownloadStart();
      // Find the ResponsiveContainer element
      const responsiveContainer = chartElement.querySelector('.recharts-responsive-container');
      if (!responsiveContainer) {
        throw new Error("Could not find chart container");
      }

      const canvas = await html2canvas(responsiveContainer as HTMLElement, {
        backgroundColor: null,
        scale: 2,
      });

      const link = document.createElement("a");
      // Use the label if available, otherwise fall back to chart-{chartId}
      const filename = label ?
        `${label.replace(/[^a-z0-9]/gi, '_').toLowerCase()}.png` :
        `chart-${chartId}.png`;
      link.download = filename;
      link.href = canvas.toDataURL("image/png", 1.0);
      link.click();
    } catch (error) {
      console.error("Error downloading chart:", error);
    } finally {
      onDownloadEnd();
    }
  };

  return (
    <button
      type="button"
      className={cx(
        "absolute top-2 right-2 z-10",
        "rounded-md p-1.5 transition-opacity",
        "bg-cbg dark:bg-dbg",
        "border border-cb dark:border-db",
        "text-ctext2 hover:text-ctext dark:text-dtext2 dark:hover:text-dtext",
        "hover:bg-cbga dark:hover:bg-dbga",
        isVisible ? "opacity-100" : "opacity-0",
      )}
      onClick={handleDownload}
      aria-label="Download chart"
    >
      <RiDownloadLine className="size-4" />
    </button>
  );
};