import React from "react";
import { ChartHoverContext } from "../contexts/ChartHoverContext";

export const ChartHoverProvider = ({
  children,
}: {
  children: React.ReactNode;
}) => {
  const [hoveredIndex, setHoveredIndex] = React.useState<
    string | number | null
  >(null);
  const [hoveredChartId, setHoveredChartId] = React.useState<string | null>(
    null,
  );

  const setHoverState = React.useCallback(
    (index: string | number | null, chartId: string | null) => {
      setHoveredIndex(index);
      setHoveredChartId(chartId);
    },
    [],
  );

  return (
    <ChartHoverContext.Provider
      value={{ hoveredIndex, hoveredChartId, setHoverState }}
    >
      <div
        className="@container flex flex-col min-h-[500px] h-full antialiased text-ctext dark:text-dtext"
        onTouchEnd={() => {
          setHoverState(null, null);
        }}
      >
        {children}
      </div>
    </ChartHoverContext.Provider>
  );
};
