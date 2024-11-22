import React from 'react';
import { ChartHoverContext } from '../contexts/ChartHoverContext';

export const ChartHoverProvider = ({ children }: { children: React.ReactNode }) => {
  const [hoveredIndex, setHoveredIndex] = React.useState<string | number | null>(null);
  const [hoveredChartId, setActiveChartId] = React.useState<string | null>(null);

  const setHoverState = React.useCallback((index: string | number | null, chartId: string | null) => {
    setHoveredIndex(index);
    setActiveChartId(chartId);
  }, []);

  return (
    <ChartHoverContext.Provider value={{ hoveredIndex, hoveredChartId, setHoverState }}>
      {children}
    </ChartHoverContext.Provider>
  );
};
