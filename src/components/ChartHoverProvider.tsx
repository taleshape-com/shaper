import React from "react";
import { ChartHoverContext } from "../contexts/ChartHoverContext";
import { Column } from "../lib/dashboard";

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
  const [hoveredIndexType, setHoveredIndexType] = React.useState<Column["type"] | null>(
    null,
  );

  const setHoverState = React.useCallback(
    (index: string | number | null, chartId: string | null, indexType: Column["type"] | null) => {
      setHoveredIndex(index);
      setHoveredChartId(chartId);
      setHoveredIndexType(indexType);
    },
    [],
  );

  return (
    <ChartHoverContext.Provider
      value={{ hoveredIndex, hoveredChartId, hoveredIndexType, setHoverState }}
    >
      <div
        className="@container flex flex-col min-h-[500px] h-full antialiased text-ctext dark:text-dtext"
        onTouchEnd={() => {
          setHoverState(null, null, null);
        }}
      >
        {children}
      </div>
    </ChartHoverContext.Provider>
  );
};
